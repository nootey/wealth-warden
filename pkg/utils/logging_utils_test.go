package utils_test

import (
	"testing"
	"time"
	"wealth-warden/pkg/utils"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestChanges_IsEmpty(t *testing.T) {
	t.Run("empty changes", func(t *testing.T) {
		c := utils.InitChanges()
		assert.True(t, c.IsEmpty())
	})

	t.Run("has new changes", func(t *testing.T) {
		c := utils.InitChanges()
		c.New["field"] = "value"
		assert.False(t, c.IsEmpty())
	})

	t.Run("has old changes", func(t *testing.T) {
		c := utils.InitChanges()
		c.Old["field"] = "value"
		assert.False(t, c.IsEmpty())
	})
}

func TestChanges_HasChanges(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		c := utils.InitChanges()
		assert.False(t, c.HasChanges())
	})

	t.Run("has changes", func(t *testing.T) {
		c := utils.InitChanges()
		c.New["field"] = "value"
		assert.True(t, c.HasChanges())
	})
}

func TestCompareChanges(t *testing.T) {
	t.Run("no change when values equal", func(t *testing.T) {
		c := utils.InitChanges()
		utils.CompareChanges("test", "test", c, "field")
		assert.True(t, c.IsEmpty())
	})

	t.Run("new value added", func(t *testing.T) {
		c := utils.InitChanges()
		utils.CompareChanges("", "new", c, "field")
		assert.Equal(t, "new", c.New["field"])
		assert.Empty(t, c.Old["field"])
	})

	t.Run("old value removed", func(t *testing.T) {
		c := utils.InitChanges()
		utils.CompareChanges("old", "", c, "field")
		assert.Equal(t, "old", c.Old["field"])
		assert.Empty(t, c.New["field"])
	})

	t.Run("value changed", func(t *testing.T) {
		c := utils.InitChanges()
		utils.CompareChanges("old", "new", c, "field")
		assert.Equal(t, "old", c.Old["field"])
		assert.Equal(t, "new", c.New["field"])
	})
}

func TestCompareDecimalChange(t *testing.T) {
	t.Run("no change", func(t *testing.T) {
		c := utils.InitChanges()
		old := decimal.NewFromFloat(10.50)
		new := decimal.NewFromFloat(10.50)
		utils.CompareDecimalChange(&old, &new, c, "amount", 2)
		assert.True(t, c.IsEmpty())
	})

	t.Run("value changed", func(t *testing.T) {
		c := utils.InitChanges()
		old := decimal.NewFromFloat(10.50)
		new := decimal.NewFromFloat(20.75)
		utils.CompareDecimalChange(&old, &new, c, "amount", 2)
		assert.Equal(t, "10.50", c.Old["amount"])
		assert.Equal(t, "20.75", c.New["amount"])
	})

	t.Run("nil to value", func(t *testing.T) {
		c := utils.InitChanges()
		new := decimal.NewFromFloat(10.50)
		utils.CompareDecimalChange(nil, &new, c, "amount", 2)
		assert.Equal(t, "10.50", c.New["amount"])
	})

	t.Run("value to nil", func(t *testing.T) {
		c := utils.InitChanges()
		old := decimal.NewFromFloat(10.50)
		utils.CompareDecimalChange(&old, nil, c, "amount", 2)
		assert.Equal(t, "10.50", c.Old["amount"])
	})
}

func TestCompareDateChange(t *testing.T) {
	t.Run("no change", func(t *testing.T) {
		c := utils.InitChanges()
		date := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
		utils.CompareDateChange(&date, &date, c, "date")
		assert.True(t, c.IsEmpty())
	})

	t.Run("date changed", func(t *testing.T) {
		c := utils.InitChanges()
		old := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		new := time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)
		utils.CompareDateChange(&old, &new, c, "date")
		assert.Equal(t, "2024-06-15", c.Old["date"])
		assert.Equal(t, "2024-06-20", c.New["date"])
	})

	t.Run("nil to date", func(t *testing.T) {
		c := utils.InitChanges()
		new := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		utils.CompareDateChange(nil, &new, c, "date")
		assert.Equal(t, "2024-06-15", c.New["date"])
	})

	t.Run("date to nil", func(t *testing.T) {
		c := utils.InitChanges()
		old := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		utils.CompareDateChange(&old, nil, c, "date")
		assert.Equal(t, "2024-06-15", c.Old["date"])
	})
}
