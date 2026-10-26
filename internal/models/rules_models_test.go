package models

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestRuleMatches(t *testing.T) {
	cond := func(field, op, value string) RuleCondition {
		return RuleCondition{Field: field, Operator: op, Value: value}
	}
	allOf := func(conds ...RuleCondition) Rule { return Rule{MatchType: "all", Conditions: conds} }
	anyOf := func(conds ...RuleCondition) Rule { return Rule{MatchType: "any", Conditions: conds} }
	// child attaches a plain condition to the group with the given id
	child := func(parent int64, c RuleCondition) RuleCondition {
		c.ParentID = &parent
		return c
	}
	amt := decimal.RequireFromString("42.50")

	cases := []struct {
		name string
		rule Rule
		dir  TransactionDirection
		want bool
	}{
		{"description contains, case-insensitive", allOf(cond("description", "contains", "spar")), "", true},
		{"description no match", allOf(cond("description", "contains", "hofer")), "", false},
		{"description with amount operator is inert", allOf(cond("description", "gt", "spar")), "", false},
		{"amount equals", allOf(cond("amount", "equals", "42.5")), "", true},
		{"amount gte", allOf(cond("amount", "gte", "42.50")), "", true},
		{"amount lt fails", allOf(cond("amount", "lt", "10")), "", false},
		{"amount non-numeric value is inert", allOf(cond("amount", "gt", "abc")), "", false},
		{"all conditions must hold", allOf(cond("description", "contains", "spar"), cond("amount", "gt", "100")), "", false},
		{"any needs one condition", anyOf(cond("description", "contains", "hofer"), cond("amount", "equals", "42.5")), "", true},
		{"any with no hit fails", anyOf(cond("description", "contains", "hofer"), cond("amount", "gt", "100")), "", false},
		{"unknown match type is inert", Rule{MatchType: "", Conditions: []RuleCondition{cond("description", "contains", "spar")}}, "", false},
		{"no conditions never match", allOf(), "", false},
		{"empty any never matches", anyOf(), "", false},
		{"group: description and (amount 2.99 or 42.50)", allOf(
			cond("description", "contains", "spar"),
			RuleCondition{ID: 7, IsGroup: true, MatchType: "any"},
			child(7, cond("amount", "equals", "2.99")),
			child(7, cond("amount", "equals", "42.50")),
		), "", true},
		{"group: all inside any", anyOf(
			cond("description", "contains", "hofer"),
			RuleCondition{ID: 7, IsGroup: true, MatchType: "all"},
			child(7, cond("description", "contains", "spar")),
			child(7, cond("amount", "gt", "100")),
		), "", false},
		{"group with unknown match type is inert", allOf(RuleCondition{ID: 7, IsGroup: true, MatchType: "x"}, child(7, cond("description", "contains", "spar"))), "", false},
		{"empty group never matches", allOf(RuleCondition{ID: 7, IsGroup: true, MatchType: "all"}), "", false},
		{"direction equals expense matches", allOf(cond("direction", "equals", "expense")), TxnDirectionExpense, true},
		{"direction equals expense fails on income", allOf(cond("direction", "equals", "expense")), TxnDirectionIncome, false},
		{"direction with gt operator is inert", allOf(cond("direction", "gt", "expense")), TxnDirectionExpense, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.Matches("SPAR LJUBLJANA - nakup", amt, tc.dir); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
