package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestRuleMatches(t *testing.T) {
	cond := func(field, op, value string) RuleCondition {
		return RuleCondition{Field: field, Operator: op, Value: value}
	}
	all := func(conds ...RuleCondition) Rule { return Rule{MatchType: "all", Conditions: conds} }
	any := func(conds ...RuleCondition) Rule { return Rule{MatchType: "any", Conditions: conds} }
	// child attaches a plain condition to the group with the given id
	child := func(parent int64, c RuleCondition) RuleCondition {
		c.ParentID = &parent
		return c
	}
	amt := decimal.RequireFromString("42.50")

	cases := []struct {
		name string
		rule Rule
		want bool
	}{
		{"description contains, case-insensitive", all(cond("description", "contains", "spar")), true},
		{"description no match", all(cond("description", "contains", "hofer")), false},
		{"description with amount operator is inert", all(cond("description", "gt", "spar")), false},
		{"amount equals", all(cond("amount", "equals", "42.5")), true},
		{"amount gte", all(cond("amount", "gte", "42.50")), true},
		{"amount lt fails", all(cond("amount", "lt", "10")), false},
		{"amount non-numeric value is inert", all(cond("amount", "gt", "abc")), false},
		{"all conditions must hold", all(cond("description", "contains", "spar"), cond("amount", "gt", "100")), false},
		{"any needs one condition", any(cond("description", "contains", "hofer"), cond("amount", "equals", "42.5")), true},
		{"any with no hit fails", any(cond("description", "contains", "hofer"), cond("amount", "gt", "100")), false},
		{"unknown match type is inert", Rule{MatchType: "", Conditions: []RuleCondition{cond("description", "contains", "spar")}}, false},
		{"no conditions never match", all(), false},
		{"empty any never matches", any(), false},
		{"group: description and (amount 2.99 or 42.50)", all(
			cond("description", "contains", "spar"),
			RuleCondition{ID: 7, IsGroup: true, MatchType: "any"},
			child(7, cond("amount", "equals", "2.99")),
			child(7, cond("amount", "equals", "42.50")),
		), true},
		{"group: all inside any", any(
			cond("description", "contains", "hofer"),
			RuleCondition{ID: 7, IsGroup: true, MatchType: "all"},
			child(7, cond("description", "contains", "spar")),
			child(7, cond("amount", "gt", "100")),
		), false},
		{"group with unknown match type is inert", all(RuleCondition{ID: 7, IsGroup: true, MatchType: "x"}, child(7, cond("description", "contains", "spar"))), false},
		{"empty group never matches", all(RuleCondition{ID: 7, IsGroup: true, MatchType: "all"}), false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.Matches("SPAR LJUBLJANA - nakup", amt); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
