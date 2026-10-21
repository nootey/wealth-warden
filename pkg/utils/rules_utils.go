package utils

import (
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
)

// MatchingRuleCategories returns the set_category target ids of every rule that
// matches, in rule order. The caller resolves which category to apply, normally
// the first that still exists. Pass only the rules to evaluate; inactive rules
// must already be filtered out by the caller.
func MatchingRuleCategories(rules []models.Rule, description string, amount decimal.Decimal, direction models.TransactionDirection) []int64 {
	var out []int64
	for _, rule := range rules {
		if !rule.Matches(description, amount, direction) {
			continue
		}
		if categoryID, ok := rule.CategoryID(); ok {
			out = append(out, categoryID)
		}
	}
	return out
}
