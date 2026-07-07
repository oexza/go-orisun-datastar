package httpui

import (
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

type rateLimitKeyFunc func(*http.Request) string

type rateLimitOptions struct {
	name     string
	window   time.Duration
	max      int
	methods  map[string]struct{}
	key      rateLimitKeyFunc
	message  string
	now      func() time.Time
	response func(http.ResponseWriter, *http.Request, string, int, time.Time)
}

type rateLimitRecord struct {
	count   int
	resetAt time.Time
}

const (
	minute = time.Minute
	hour   = time.Hour
)

var (
	loginRateLimit = rateLimit(rateLimitOptions{
		name:    "login",
		window:  15 * minute,
		max:     envPositiveInt("LOGIN_RATE_LIMIT_MAX", 10),
		methods: methods(http.MethodPost),
	})
	signupRateLimit = rateLimit(rateLimitOptions{
		name:    "signup",
		window:  hour,
		max:     envPositiveInt("SIGNUP_RATE_LIMIT_MAX", 5),
		methods: methods(http.MethodPost),
	})
	forgotPasswordRateLimit = rateLimit(rateLimitOptions{
		name:    "forgot-password",
		window:  15 * minute,
		max:     envPositiveInt("PASSWORD_RESET_REQUEST_RATE_LIMIT_MAX", 5),
		methods: methods(http.MethodPost),
	})
	resetPasswordRateLimit = rateLimit(rateLimitOptions{
		name:    "reset-password",
		window:  15 * minute,
		max:     envPositiveInt("PASSWORD_RESET_SUBMIT_RATE_LIMIT_MAX", 8),
		methods: methods(http.MethodPost),
	})
	otpResendRateLimit = rateLimit(rateLimitOptions{
		name:    "otp-resend",
		window:  15 * minute,
		max:     envPositiveInt("OTP_RESEND_RATE_LIMIT_MAX", 3),
		methods: methods(http.MethodPost),
		key:     ipAndURLParamKey("userID"),
	})
	otpValidateRateLimit = rateLimit(rateLimitOptions{
		name:    "otp-validate",
		window:  15 * minute,
		max:     envPositiveInt("OTP_VALIDATE_RATE_LIMIT_MAX", 10),
		methods: methods(http.MethodPost),
		key:     ipAndURLParamKey("userID"),
	})
)

func envPositiveInt(name string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func rateLimit(opts rateLimitOptions) func(http.Handler) http.Handler {
	if opts.max <= 0 {
		opts.max = 1
	}
	if opts.window <= 0 {
		opts.window = time.Minute
	}
	if opts.key == nil {
		opts.key = ipKey
	}
	if opts.message == "" {
		opts.message = "Too many requests. Please wait a moment and try again."
	}
	if opts.now == nil {
		opts.now = time.Now
	}
	if opts.response == nil {
		opts.response = writeRateLimitExceeded
	}

	var (
		mu          sync.Mutex
		records     = map[string]rateLimitRecord{}
		nextSweepAt = opts.now().Add(opts.window)
	)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(opts.methods) > 0 {
				if _, ok := opts.methods[strings.ToUpper(r.Method)]; !ok {
					next.ServeHTTP(w, r)
					return
				}
			}

			now := opts.now()
			key := opts.name + ":" + opts.key(r)

			mu.Lock()
			if !now.Before(nextSweepAt) {
				for recordKey, record := range records {
					if !record.resetAt.After(now) {
						delete(records, recordKey)
					}
				}
				nextSweepAt = now.Add(opts.window)
			}

			record, ok := records[key]
			if !ok || !record.resetAt.After(now) {
				records[key] = rateLimitRecord{count: 1, resetAt: now.Add(opts.window)}
				mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			if record.count >= opts.max {
				resetAt := record.resetAt
				mu.Unlock()
				opts.response(w, r, opts.message, opts.max, resetAt)
				return
			}

			record.count++
			records[key] = record
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func methods(methods ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		out[strings.ToUpper(method)] = struct{}{}
	}
	return out
}

func ipKey(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); value != "" {
		return value
	}
	if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
		return strings.TrimSpace(strings.Split(value, ",")[0])
	}
	if value := strings.TrimSpace(r.Header.Get("X-Real-IP")); value != "" {
		return value
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}

func ipAndURLParamKey(param string) rateLimitKeyFunc {
	return func(r *http.Request) string {
		return ipKey(r) + ":" + param + ":" + strings.TrimSpace(chi.URLParam(r, param))
	}
}

func writeRateLimitExceeded(w http.ResponseWriter, _ *http.Request, message string, limit int, resetAt time.Time) {
	retryAfter := int(time.Until(resetAt).Seconds())
	if retryAfter < 1 {
		retryAfter = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.Header().Set("X-RateLimit-Reset", resetAt.UTC().Format(time.RFC3339))
	http.Error(w, message, http.StatusTooManyRequests)
}
