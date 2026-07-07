package commandlimits

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const defaultStringLimit = 4096

type fieldLimit struct {
	pattern *regexp.Regexp
	limit   int
}

var fieldLimits = []fieldLimit{
	{regexp.MustCompile(`(?i)^(.*\.)?(id|.*id|.*ids|.*createdid|.*registeredid|.*requestedid|.*generatedid|.*uploadedid|.*deletedid|.*updatedid|.*reopenedid|.*completedid|.*renamedid|eventid|token)$`), 128},
	{regexp.MustCompile(`(?i)^(.*\.)?(email|emailaddress)$`), 320},
	{regexp.MustCompile(`(?i)^(.*\.)?(username)$`), 32},
	{regexp.MustCompile(`(?i)^(.*\.)?(password|currentpassword|newpassword|confirmpassword)$`), 1024},
	{regexp.MustCompile(`(?i)^(.*\.)?(name|firstname|lastname|title)$`), 160},
	{regexp.MustCompile(`(?i)^(.*\.)?(contenttype)$`), 100},
	{regexp.MustCompile(`(?i)^(.*\.)?(host|ipaddress|useragent|acceptlanguage|forwardedfor|remoteaddr)$`), 512},
	{regexp.MustCompile(`(?i)^(.*\.)?(referrer|referer|origin|path)$`), 2048},
	{regexp.MustCompile(`(?i)^(.*\.)?(otp|code)$`), 16},
	{regexp.MustCompile(`(?i)^(.*\.)?(bio)$`), 280},
	{regexp.MustCompile(`(?i)^(.*\.)?(imageurl|headerimageurl|url)$`), 2048},
}

type LimitError struct {
	Field  string
	Limit  int
	Actual int
}

func (e LimitError) Error() string {
	return fmt.Sprintf("command input %q exceeds %d characters", e.Field, e.Limit)
}

func Assert(input any) error {
	seen := map[uintptr]struct{}{}
	return assertValue(reflect.ValueOf(input), "command", seen)
}

func assertValue(value reflect.Value, path string, seen map[uintptr]struct{}) error {
	if !value.IsValid() {
		return nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		if value.Kind() == reflect.Pointer {
			ptr := value.Pointer()
			if _, ok := seen[ptr]; ok {
				return nil
			}
			seen[ptr] = struct{}{}
		}
		value = value.Elem()
	}

	if value.Type() == reflect.TypeOf(time.Time{}) {
		return nil
	}

	switch value.Kind() {
	case reflect.String:
		limit := limitForPath(path)
		actual := utf8.RuneCountInString(value.String())
		if actual > limit {
			return LimitError{Field: path, Limit: limit, Actual: actual}
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			if err := assertValue(value.Field(i), path+"."+field.Name, seen); err != nil {
				return err
			}
		}
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			key := fmt.Sprint(iter.Key().Interface())
			key = strings.TrimSpace(key)
			if key == "" {
				key = "value"
			}
			if err := assertValue(iter.Value(), path+"."+key, seen); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		if value.Type().Elem().Kind() == reflect.Uint8 {
			return nil
		}
		for i := 0; i < value.Len(); i++ {
			if err := assertValue(value.Index(i), fmt.Sprintf("%s.%d", path, i), seen); err != nil {
				return err
			}
		}
	}
	return nil
}

func limitForPath(path string) int {
	normalized := regexp.MustCompile(`(?:\.\d+)+$`).ReplaceAllString(path, "")
	for _, entry := range fieldLimits {
		if entry.pattern.MatchString(normalized) {
			return entry.limit
		}
	}
	return defaultStringLimit
}
