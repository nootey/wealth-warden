package utils_test

import (
	"testing"
	"wealth-warden/pkg/utils"

	"github.com/stretchr/testify/assert"
)

func TestSafeString(t *testing.T) {
	t.Run("returns empty string for nil pointer", func(t *testing.T) {
		var s *string = nil

		result := utils.SafeString(s)

		assert.Equal(t, "", result)
	})

	t.Run("returns trimmed string for non-nil pointer", func(t *testing.T) {
		s := "  hello world  "

		result := utils.SafeString(&s)

		assert.Equal(t, "hello world", result)
	})

	t.Run("returns empty string for pointer to empty string", func(t *testing.T) {
		s := ""

		result := utils.SafeString(&s)

		assert.Equal(t, "", result)
	})

	t.Run("returns trimmed string for pointer to whitespace-only string", func(t *testing.T) {
		s := "   "

		result := utils.SafeString(&s)

		assert.Equal(t, "", result)
	})

	t.Run("handles string with no whitespace", func(t *testing.T) {
		s := "hello"

		result := utils.SafeString(&s)

		assert.Equal(t, "hello", result)
	})
}

func TestCleanString(t *testing.T) {
	t.Run("trims string type", func(t *testing.T) {
		input := "  hello world  "

		result := utils.CleanString(input)

		assert.Equal(t, "hello world", result)
	})

	t.Run("trims pointer to string", func(t *testing.T) {
		s := "  hello world  "
		input := &s

		result := utils.CleanString(input)

		assert.IsType(t, (*string)(nil), result)
		resultPtr := result.(*string)
		assert.Equal(t, "hello world", *resultPtr)
	})

	t.Run("returns nil for nil string pointer", func(t *testing.T) {
		var input *string = nil

		result := utils.CleanString(input)

		assert.Nil(t, result)
	})

	t.Run("returns empty string for empty input", func(t *testing.T) {
		input := ""

		result := utils.CleanString(input)

		assert.Equal(t, "", result)
	})

	t.Run("trims whitespace-only string", func(t *testing.T) {
		input := "   \t\n   "

		result := utils.CleanString(input)

		assert.Equal(t, "", result)
	})

	t.Run("trims pointer to whitespace-only string", func(t *testing.T) {
		s := "   \t\n   "
		input := &s

		result := utils.CleanString(input)

		resultPtr := result.(*string)
		assert.Equal(t, "", *resultPtr)
	})

	t.Run("preserves internal spaces", func(t *testing.T) {
		input := "  hello   world  "

		result := utils.CleanString(input)

		assert.Equal(t, "hello   world", result)
	})

	t.Run("returns unchanged for non-string types", func(t *testing.T) {
		testCases := []interface{}{
			123,
			true,
			[]string{"test"},
			map[string]string{"key": "value"},
		}

		for _, input := range testCases {
			result := utils.CleanString(input)
			assert.Equal(t, input, result)
		}
	})

	t.Run("handles string with leading whitespace only", func(t *testing.T) {
		input := "  hello"

		result := utils.CleanString(input)

		assert.Equal(t, "hello", result)
	})

	t.Run("handles string with trailing whitespace only", func(t *testing.T) {
		input := "hello  "

		result := utils.CleanString(input)

		assert.Equal(t, "hello", result)
	})

	t.Run("handles pointer to string with no whitespace", func(t *testing.T) {
		s := "hello"
		input := &s

		result := utils.CleanString(input)

		resultPtr := result.(*string)
		assert.Equal(t, "hello", *resultPtr)
	})
}

func TestNormalizeName(t *testing.T) {
	t.Run("converts to lowercase", func(t *testing.T) {
		input := "Hello World"

		result := utils.NormalizeName(input)

		assert.Equal(t, "hello_world", result)
	})

	t.Run("replaces spaces with underscores", func(t *testing.T) {
		input := "my test name"

		result := utils.NormalizeName(input)

		assert.Equal(t, "my_test_name", result)
	})

	t.Run("replaces colons with underscores", func(t *testing.T) {
		input := "test:name:here"

		result := utils.NormalizeName(input)

		assert.Equal(t, "test_name_here", result)
	})

	t.Run("handles mixed spaces and colons", func(t *testing.T) {
		input := "Test Name:Value"

		result := utils.NormalizeName(input)

		assert.Equal(t, "test_name_value", result)
	})

	t.Run("handles already normalized string", func(t *testing.T) {
		input := "test_name"

		result := utils.NormalizeName(input)

		assert.Equal(t, "test_name", result)
	})

	t.Run("handles empty string", func(t *testing.T) {
		input := ""

		result := utils.NormalizeName(input)

		assert.Equal(t, "", result)
	})

	t.Run("handles string with multiple consecutive spaces", func(t *testing.T) {
		input := "hello    world"

		result := utils.NormalizeName(input)

		assert.Equal(t, "hello____world", result)
	})

	t.Run("handles string with multiple consecutive colons", func(t *testing.T) {
		input := "test:::name"

		result := utils.NormalizeName(input)

		assert.Equal(t, "test___name", result)
	})

	t.Run("handles uppercase with special chars", func(t *testing.T) {
		input := "USER NAME:123"

		result := utils.NormalizeName(input)

		assert.Equal(t, "user_name_123", result)
	})

	t.Run("preserves other special characters", func(t *testing.T) {
		input := "test-name.value"

		result := utils.NormalizeName(input)

		assert.Equal(t, "test-name.value", result)
	})

	t.Run("handles single character", func(t *testing.T) {
		input := "A"

		result := utils.NormalizeName(input)

		assert.Equal(t, "a", result)
	})
}
