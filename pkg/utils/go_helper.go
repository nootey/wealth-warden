package utils

import (
	"reflect"
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

func MapJSONToStructField(jsonField string, modelType interface{}) (string, bool) {
	t := reflect.TypeOf(modelType)

	// Iterate through struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")

		// JSON tags can contain ",omitempty", so split on "," to get the actual name
		if jsonTag != "" {
			jsonTag = strings.Split(jsonTag, ",")[0]
		}

		if jsonTag == jsonField {
			return field.Name, true
		}
	}
	return "", false
}
