package utils

import (
	"strconv"
	"strings"
)

func SafeString(s *string) string {
	if s == nil {
		return ""
	}
	return CleanString(*s).(string)
}

// CleanString trims leading/trailing spaces for both `string` and `*string` types.
// - If a string is passed, it returns a cleaned string.
// - If a *string is passed, it returns a cleaned *string (or nil if the input was nil).
func CleanString(input interface{}) interface{} {
	switch v := input.(type) {
	case string:
		return strings.TrimSpace(v)
	case *string:
		if v == nil {
			return nil
		}
		cleaned := strings.TrimSpace(*v)
		return &cleaned
	default:
		return input
	}
}

func StrToUint(s string) (uint, error) {
	number, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	uintNumber := uint(number)

	return uintNumber, nil
}

func UintToStr(u uint) (string, error) {
	str := strconv.FormatUint(uint64(u), 10)
	return str, nil
}
