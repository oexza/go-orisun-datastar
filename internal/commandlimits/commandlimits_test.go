package commandlimits

import (
	"errors"
	"strings"
	"testing"
)

func TestAssertUsesFieldSpecificLimit(t *testing.T) {
	err := Assert(struct {
		Email string
	}{Email: strings.Repeat("a", 321)})
	if err == nil {
		t.Fatal("expected limit error")
	}
	var limitErr LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected LimitError, got %T", err)
	}
	if limitErr.Field != "command.Email" || limitErr.Limit != 320 || limitErr.Actual != 321 {
		t.Fatalf("unexpected limit error: %+v", limitErr)
	}
}

func TestAssertSkipsByteSlices(t *testing.T) {
	err := Assert(struct {
		Data []byte
	}{Data: make([]byte, 1024*1024)})
	if err != nil {
		t.Fatalf("expected byte slice to be skipped, got %v", err)
	}
}

func TestAssertWalksNestedMaps(t *testing.T) {
	err := Assert(map[string]any{
		"audit": map[string]any{
			"userAgent": strings.Repeat("x", 513),
		},
	})
	if err == nil {
		t.Fatal("expected nested map limit error")
	}
	var limitErr LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected LimitError, got %T", err)
	}
	if limitErr.Field != "command.audit.userAgent" || limitErr.Limit != 512 {
		t.Fatalf("unexpected nested limit error: %+v", limitErr)
	}
}
