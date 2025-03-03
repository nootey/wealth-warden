package utils

import "strings"

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
