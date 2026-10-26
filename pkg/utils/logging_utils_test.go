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

func TestChanges_Stamp(t *testing.T) {
	t.Run("identical record still reports no changes", func(t *testing.T) {
		c := utils.InitChanges()
		utils.CompareChanges("Groceries", "Groceries", c, "category")
		utils.CompareChanges("EUR", "EUR", c, "currency")
		assert.False(t, c.HasChanges())
	})

	t.Run("stamp adds identifier without affecting the diff", func(t *testing.T) {
		c := utils.InitChanges()
		utils.CompareChanges("Groceries", "Rent", c, "category")
		assert.True(t, c.HasChanges())

		c.Stamp("id", "42")
		assert.Equal(t, "42", c.New["id"])
		assert.Equal(t, "Rent", c.New["category"])
		assert.Equal(t, "Groceries", c.Old["category"])
		assert.Empty(t, c.Old["id"])
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
		oldV := decimal.NewFromFloat(10.50)
		newV := decimal.NewFromFloat(10.50)
		utils.CompareDecimalChange(&oldV, &newV, c, "amount", 2)
		assert.True(t, c.IsEmpty())
	})

	t.Run("value changed", func(t *testing.T) {
		c := utils.InitChanges()
		oldV := decimal.NewFromFloat(10.50)
		newV := decimal.NewFromFloat(20.75)
		utils.CompareDecimalChange(&oldV, &newV, c, "amount", 2)
		assert.Equal(t, "10.50", c.Old["amount"])
		assert.Equal(t, "20.75", c.New["amount"])
	})

	t.Run("nil to value", func(t *testing.T) {
		c := utils.InitChanges()
		newV := decimal.NewFromFloat(10.50)
		utils.CompareDecimalChange(nil, &newV, c, "amount", 2)
		assert.Equal(t, "10.50", c.New["amount"])
	})

	t.Run("value to nil", func(t *testing.T) {
		c := utils.InitChanges()
		oldV := decimal.NewFromFloat(10.50)
		utils.CompareDecimalChange(&oldV, nil, c, "amount", 2)
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
		oldV := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		newV := time.Date(2024, 6, 20, 0, 0, 0, 0, time.UTC)
		utils.CompareDateChange(&oldV, &newV, c, "date")
		assert.Equal(t, "2024-06-15", c.Old["date"])
		assert.Equal(t, "2024-06-20", c.New["date"])
	})

	t.Run("nil to date", func(t *testing.T) {
		c := utils.InitChanges()
		newV := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		utils.CompareDateChange(nil, &newV, c, "date")
		assert.Equal(t, "2024-06-15", c.New["date"])
	})

	t.Run("date to nil", func(t *testing.T) {
		c := utils.InitChanges()
		oldV := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		utils.CompareDateChange(&oldV, nil, c, "date")
		assert.Equal(t, "2024-06-15", c.Old["date"])
	})
}
