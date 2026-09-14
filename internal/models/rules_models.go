package models

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const (
	RuleFieldDescription = "description"
	RuleFieldAmount      = "amount"

	RuleOpContains = "contains"
	RuleOpEquals   = "equals"
	RuleOpGt       = "gt"
	RuleOpGte      = "gte"
	RuleOpLt       = "lt"
	RuleOpLte      = "lte"

	RuleActionSetCategory = "set_category"
)

type Rule struct {
	ID            int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID        int64           `gorm:"not null;index:idx_rules_user_id" json:"user_id"`
	Name          string          `gorm:"type:varchar(100);not null" json:"name"`
	IsActive      bool            `gorm:"not null" json:"is_active"`
	EffectiveDate *time.Time      `gorm:"type:date" json:"effective_date,omitempty"`
	Conditions    []RuleCondition `gorm:"foreignKey:RuleID" json:"conditions"`
	Actions       []RuleAction    `gorm:"foreignKey:RuleID" json:"actions"`
	CreatedAt     time.Time       `gorm:"autoCreateTime;not null" json:"created_at"`
	UpdatedAt     time.Time       `gorm:"autoUpdateTime;not null" json:"updated_at"`
}

type RuleCondition struct {
	ID        int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID    int64  `gorm:"not null;index:idx_rule_conditions_rule_id" json:"rule_id"`
	ParentID  *int64 `json:"parent_id,omitempty"`
	IsGroup   bool   `gorm:"not null;default:false" json:"is_group"`
	MatchType string `gorm:"type:varchar(10);not null" json:"match_type"`
	Field     string `gorm:"type:varchar(50);not null" json:"field"`
	Operator  string `gorm:"type:varchar(20);not null" json:"operator"`
	Value     string `gorm:"type:text;not null" json:"value"`
	Position  int    `gorm:"not null;default:0" json:"position"`
}

type RuleAction struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	RuleID     int64  `gorm:"not null;index:idx_rule_actions_rule_id" json:"rule_id"`
	ActionType string `gorm:"type:varchar(50);not null" json:"action_type"`
	Value      string `gorm:"type:text;not null" json:"value"`
	Position   int    `gorm:"not null;default:0" json:"position"`
}

// Matches reports whether every plain condition holds. Groups and unknown fields never match,
// so a rule the current code cannot evaluate stays inert instead of mis-categorizing.
func (r Rule) Matches(description string, amount decimal.Decimal) bool {
	if len(r.Conditions) == 0 {
		return false
	}
	for _, c := range r.Conditions {
		if !c.matches(description, amount) {
			return false
		}
	}
	return true
}

func (c RuleCondition) matches(description string, amount decimal.Decimal) bool {
	if c.IsGroup {
		return false
	}
	switch c.Field {
	case RuleFieldDescription:
		if c.Operator != RuleOpContains {
			return false
		}
		return strings.Contains(strings.ToLower(description), strings.ToLower(c.Value))
	case RuleFieldAmount:
		want, err := decimal.NewFromString(c.Value)
		if err != nil {
			return false
		}
		switch c.Operator {
		case RuleOpEquals:
			return amount.Equal(want)
		case RuleOpGt:
			return amount.GreaterThan(want)
		case RuleOpGte:
			return amount.GreaterThanOrEqual(want)
		case RuleOpLt:
			return amount.LessThan(want)
		case RuleOpLte:
			return amount.LessThanOrEqual(want)
		}
	}
	return false
}

func (r Rule) CategoryID() (int64, bool) {
	for _, a := range r.Actions {
		if a.ActionType == RuleActionSetCategory {
			id, err := decimal.NewFromString(a.Value)
			if err != nil || !id.IsInteger() {
				return 0, false
			}
			return id.IntPart(), true
		}
	}
	return 0, false
}

type RuleReq struct {
	Name       string             `json:"name" validate:"max=100"`
	IsActive   *bool              `json:"is_active"`
	Conditions []RuleConditionReq `json:"conditions" validate:"required,min=1,max=1,dive"`
	Actions    []RuleActionReq    `json:"actions" validate:"required,min=1,max=1,dive"`
}

type RuleConditionReq struct {
	Field    string `json:"field" validate:"required,oneof=description amount"`
	Operator string `json:"operator" validate:"required,oneof=contains equals gt gte lt lte"`
	Value    string `json:"value" validate:"required,max=255"`
}

type RuleActionReq struct {
	ActionType string `json:"action_type" validate:"required,oneof=set_category"`
	Value      string `json:"value" validate:"required,max=255"`
}
