package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestRuleMatches(t *testing.T) {
	cond := func(field, op, value string) RuleCondition {
		return RuleCondition{Field: field, Operator: op, Value: value}
	}
	amt := decimal.RequireFromString("42.50")

	cases := []struct {
		name string
		rule Rule
		want bool
	}{
		{"description contains, case-insensitive", Rule{Conditions: []RuleCondition{cond("description", "contains", "spar")}}, true},
		{"description no match", Rule{Conditions: []RuleCondition{cond("description", "contains", "hofer")}}, false},
		{"description with amount operator is inert", Rule{Conditions: []RuleCondition{cond("description", "gt", "spar")}}, false},
		{"amount equals", Rule{Conditions: []RuleCondition{cond("amount", "equals", "42.5")}}, true},
		{"amount gte", Rule{Conditions: []RuleCondition{cond("amount", "gte", "42.50")}}, true},
		{"amount lt fails", Rule{Conditions: []RuleCondition{cond("amount", "lt", "10")}}, false},
		{"amount non-numeric value is inert", Rule{Conditions: []RuleCondition{cond("amount", "gt", "abc")}}, false},
		{"all conditions must hold", Rule{Conditions: []RuleCondition{cond("description", "contains", "spar"), cond("amount", "gt", "100")}}, false},
		{"group is inert", Rule{Conditions: []RuleCondition{{IsGroup: true, MatchType: "any"}}}, false},
		{"no conditions never match", Rule{}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.Matches("SPAR LJUBLJANA - nakup", amt); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
