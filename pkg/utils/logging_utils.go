package utils

import (
	"time"

	"github.com/shopspring/decimal"
)

type Changes struct {
	New map[string]string
	Old map[string]string
}

func (c *Changes) IsEmpty() bool {
	return len(c.New) == 0 && len(c.Old) == 0
}

func InitChanges() *Changes {
	return &Changes{
		New: make(map[string]string),
		Old: make(map[string]string),
	}
}

func CompareChanges(old, new string, obj *Changes, index string) {
	if old != new {
		if len(new) == 0 {
			obj.Old[index] = old
		} else if len(old) == 0 {
			obj.New[index] = new
		} else {
			obj.Old[index] = old
			obj.New[index] = new
		}
	}
}

func CompareDecimalChange(oldV *decimal.Decimal, newV *decimal.Decimal, changes *Changes, key string, places int32) {
	var oldStr, newStr string
	if oldV != nil {
		oldStr = oldV.StringFixed(places)
	}
	if newV != nil {
		newStr = newV.StringFixed(places)
	}
	CompareChanges(oldStr, newStr, changes, key)
}

func CompareDateChange(oldT *time.Time, newT *time.Time, changes *Changes, key string) {
	var oldStr, newStr string
	if oldT != nil {
		t := oldT.UTC()
		oldStr = t.Format("2006-01-02")
	}
	if newT != nil {
		t := newT.UTC()
		newStr = t.Format("2006-01-02")
	}
	CompareChanges(oldStr, newStr, changes, key)
}

func (c *Changes) HasChanges() bool {
	return len(c.New) > 0 || len(c.Old) > 0
}
