package finance_test

import (
	"testing"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/finance"
)

func TestBuildSymbol(t *testing.T) {
	cases := []struct {
		name  string
		parts finance.SymbolParts
		typ   models.InvestmentType
		want  string
	}{
		{"crypto defaults to USD", finance.SymbolParts{Ticker: "btc"}, models.InvestmentCrypto, "BTC-USD"},
		{"crypto keeps explicit pair", finance.SymbolParts{Ticker: "BTC", Currency: "eur"}, models.InvestmentCrypto, "BTC-EUR"},
		{"US stock stays bare", finance.SymbolParts{Ticker: "aapl"}, models.InvestmentStock, "AAPL"},
		{"stock with alias resolves suffix", finance.SymbolParts{Ticker: "IWDA", Exchange: "amsterdam"}, models.InvestmentETF, "IWDA.AS"},
		{"stock with suffix code", finance.SymbolParts{Ticker: "IWDA", Exchange: "AS"}, models.InvestmentETF, "IWDA.AS"},
		{"empty ticker", finance.SymbolParts{}, models.InvestmentStock, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := finance.BuildSymbol(c.parts, c.typ); got != c.want {
				t.Errorf("BuildSymbol(%+v) = %q, want %q", c.parts, got, c.want)
			}
		})
	}
}

func TestParseSymbol(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		typ  models.InvestmentType
		want finance.SymbolParts
	}{
		{"crypto pair", "BTC-USD", models.InvestmentCrypto, finance.SymbolParts{Ticker: "BTC", Currency: "USD"}},
		{"crypto bare", "btc", models.InvestmentCrypto, finance.SymbolParts{Ticker: "BTC"}},
		{"stock suffixed", "iwda.as", models.InvestmentETF, finance.SymbolParts{Ticker: "IWDA", Exchange: "AS"}},
		{"stock bare", "AAPL", models.InvestmentStock, finance.SymbolParts{Ticker: "AAPL"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := finance.ParseSymbol(c.raw, c.typ)
			if got != c.want {
				t.Errorf("ParseSymbol(%q) = %+v, want %+v", c.raw, got, c.want)
			}
		})
	}
}

func TestNormalizeTicker(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		typ     models.InvestmentType
		want    string
		wantErr bool
	}{
		{"crypto adds USD", "btc", models.InvestmentCrypto, "BTC-USD", false},
		{"crypto keeps pair", "BTC-USDT", models.InvestmentCrypto, "BTC-USDT", false},
		{"US stock", "aapl", models.InvestmentStock, "AAPL", false},
		{"suffixed ETF", "iwda.as", models.InvestmentETF, "IWDA.AS", false},
		{"unknown exchange", "IWDA.ZZ", models.InvestmentETF, "", true},
		{"malformed stock", "not a ticker", models.InvestmentStock, "", true},
		{"empty", "", models.InvestmentCrypto, "", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := finance.NormalizeTicker(c.raw, c.typ)
			if c.wantErr {
				if err == nil {
					t.Errorf("NormalizeTicker(%q) = %q, want error", c.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeTicker(%q) unexpected error: %v", c.raw, err)
			}
			if got != c.want {
				t.Errorf("NormalizeTicker(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}
