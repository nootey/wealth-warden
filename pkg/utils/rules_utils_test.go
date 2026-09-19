package utils_test

import (
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func descRule(name, needle string, categoryID string) models.Rule {
	return models.Rule{
		Name:      name,
		IsActive:  true,
		MatchType: models.RuleMatchAll,
		Conditions: []models.RuleCondition{
			{MatchType: models.RuleMatchAll, Field: models.RuleFieldDescription, Operator: models.RuleOpContains, Value: needle},
		},
		Actions: []models.RuleAction{
			{ActionType: models.RuleActionSetCategory, Value: categoryID},
		},
	}
}

func TestMatchingRuleCategories(t *testing.T) {
	amount := decimal.NewFromInt(10)

	t.Run("no rules match", func(t *testing.T) {
		rules := []models.Rule{descRule("coffee", "starbucks", "7")}
		got := utils.MatchingRuleCategories(rules, "grocery store", amount, models.TxnDirectionExpense)
		assert.Empty(t, got)
	})

	t.Run("returns matches in rule order", func(t *testing.T) {
		rules := []models.Rule{
			descRule("first", "coffee", "3"),
			descRule("second", "shop", "5"),
		}
		got := utils.MatchingRuleCategories(rules, "Coffee Shop", amount, models.TxnDirectionExpense)
		assert.Equal(t, []int64{3, 5}, got)
	})

	t.Run("skips a matching rule without a category action", func(t *testing.T) {
		noAction := descRule("no-cat", "coffee", "9")
		noAction.Actions = nil
		rules := []models.Rule{noAction, descRule("with-cat", "coffee", "4")}
		got := utils.MatchingRuleCategories(rules, "coffee", amount, models.TxnDirectionExpense)
		assert.Equal(t, []int64{4}, got)
	})
}
