package workers

import (
	"math/rand"
	"sort"
	"testing"
	"time"
	"wealth-warden/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The deltas are the only thing keeping seeded balances honest: nothing
// recomputes them from the transactions afterwards.
func TestBulkTransactionsForAccount(t *testing.T) {
	today := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	opened := today.AddDate(-2, 0, 0)
	incCats := []int64{1, 2, 3}
	expCats := []int64{4, 5, 6}

	cases := []struct {
		name string
		seed bulkAccountSeed
	}{
		{"asset", bulkAccountSeeds[0]},
		{"liability", bulkAccountSeeds[2]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.name == "liability", tc.seed.StartBalance.IsNegative())

			acc := models.Account{ID: 7, UserID: 3, Currency: "EUR"}
			rng := rand.New(rand.NewSource(1))

			txns, deltas := bulkTransactionsForAccount(
				rng, today, opened, acc, tc.seed, 400, incCats, expCats)

			require.NotEmpty(t, txns)
			require.NotEmpty(t, deltas)

			perDay := map[time.Time]decimal.Decimal{}
			for _, txn := range txns {
				assert.False(t, txn.TxnDate.Before(opened), "date before the account opened")
				assert.False(t, txn.TxnDate.After(today), "date in the future")
				assert.True(t, txn.Amount.IsPositive(), "amount must be positive")

				signed := txn.Amount
				if txn.TransactionType == "expense" {
					signed = signed.Neg()
				}
				perDay[txn.TxnDate] = perDay[txn.TxnDate].Add(signed)
			}

			assert.Len(t, deltas, len(perDay), "one delta per day with activity")

			for _, d := range deltas {
				want, ok := perDay[d.AsOf]
				require.True(t, ok, "delta on a day with no transactions: %s", d.AsOf)
				assert.Equal(t, want.String(), d.Inflows.Sub(d.Outflows).String(),
					"delta does not match its transactions on %s", d.AsOf)
			}

			sort.Slice(deltas, func(i, j int) bool { return deltas[i].AsOf.Before(deltas[j].AsOf) })
			balance := tc.seed.StartBalance
			for _, d := range deltas {
				balance = balance.Add(d.Inflows).Sub(d.Outflows)
				if tc.seed.StartBalance.IsNegative() {
					assert.False(t, balance.IsPositive(), "liability turned positive on %s", d.AsOf)
				} else {
					assert.False(t, balance.IsNegative(), "asset went negative on %s", d.AsOf)
				}
			}
		})
	}
}

func TestBulkEmailIsStable(t *testing.T) {
	assert.Equal(t, "bulk00001@local.seed", bulkEmail(1, "local.seed"))
	assert.Equal(t, "bulk10000@local.seed", bulkEmail(10000, "local.seed"))
}

// The cost basis a seeded buy leaves on the asset has to match what the trade
// service would compute, or read-time PnL is wrong from day one.
func TestBulkPositionFromSpend(t *testing.T) {
	t.Run("crypto carries the fee in coin units", func(t *testing.T) {
		price := decimal.NewFromInt(50000)
		spend := decimal.NewFromInt(10000)

		qty, valueAtBuy, avg, fee, ok := bulkPositionFromSpend(true, price, spend, decimal.Zero)
		require.True(t, ok)

		gross := spend.Div(price).RoundDown(6) // 0.2
		wantFee := gross.Mul(decimal.NewFromFloat(0.001)).RoundDown(8)
		assert.Equal(t, wantFee.String(), fee.String())
		assert.Equal(t, gross.Sub(wantFee).String(), qty.String())
		assert.Equal(t, qty.Mul(price).String(), valueAtBuy.String())
		assert.Equal(t, price.String(), avg.String())
		assert.False(t, qty.Add(fee).GreaterThan(gross), "quantity + fee exceeds what was bought")
	})

	t.Run("stock buys whole shares with the fee apart", func(t *testing.T) {
		price := decimal.NewFromInt(150)
		spend := decimal.NewFromInt(1000)
		stockFee := decimal.NewFromFloat(2)

		qty, valueAtBuy, avg, fee, ok := bulkPositionFromSpend(false, price, spend, stockFee)
		require.True(t, ok)

		assert.Equal(t, "6", qty.String(), "1000/150 rounds down to 6 shares")
		assert.True(t, qty.Equal(qty.Truncate(0)), "must be whole shares")
		assert.Equal(t, "900", valueAtBuy.String())
		assert.Equal(t, "150", avg.String())
		assert.Equal(t, stockFee.String(), fee.String())
		assert.False(t, valueAtBuy.GreaterThan(spend), "spent more than the budget")
	})

	t.Run("a budget below one share still buys one", func(t *testing.T) {
		qty, valueAtBuy, _, _, ok := bulkPositionFromSpend(
			false, decimal.NewFromInt(5000), decimal.NewFromInt(1000), decimal.NewFromFloat(1))
		require.True(t, ok)
		assert.Equal(t, "1", qty.String())
		assert.Equal(t, "5000", valueAtBuy.String())
	})

	t.Run("no price means no position", func(t *testing.T) {
		_, _, _, _, ok := bulkPositionFromSpend(true, decimal.Zero, decimal.NewFromInt(100), decimal.Zero)
		assert.False(t, ok)
	})
}
