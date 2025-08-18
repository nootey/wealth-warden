package utils

import (
	"github.com/shopspring/decimal"
	"time"
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

func CompareDecimalChange(oldV, newV decimal.Decimal, changes *Changes, key string, places int32) {
	if !oldV.Equal(newV) {
		CompareChanges(oldV.StringFixed(places), newV.StringFixed(places), changes, key)
	}
}

func CompareDateChange(oldT, newT time.Time, changes *Changes, key string) {
	oldDate := oldT.UTC().Format("2006-01-02")
	newDate := newT.UTC().Format("2006-01-02")
	if oldDate != newDate {
		CompareChanges(oldDate, newDate, changes, key)
	}
}

func (c *Changes) HasChanges() bool {
	return len(c.New) > 0 || len(c.Old) > 0
}
