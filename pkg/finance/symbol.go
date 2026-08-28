package finance

import (
	"fmt"
	"regexp"
	"strings"
	"wealth-warden/internal/models"
)

// stockSymbolRegex accepts a US ticker (AAPL) or TICKER.EXCHANGE (IWDA.AS). The
// exchange alternation is built from ExchangeMap so the two stay in sync.
var stockSymbolRegex = func() *regexp.Regexp {
	seen := make(map[string]struct{})
	codes := make([]string, 0)
	for _, code := range ExchangeMap {
		if _, ok := seen[code]; !ok {
			seen[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	return regexp.MustCompile(`^[A-Z]{1,7}(\.(` + strings.Join(codes, "|") + `))?$`)
}()

// NormalizeExchange upper-cases an exchange name or alias and resolves it to its
// Yahoo Finance suffix code. An empty input returns "".
func NormalizeExchange(exchange string) string {
	normalized := strings.ToUpper(strings.TrimSpace(exchange))
	if normalized == "" {
		return ""
	}
	if code, ok := ExchangeMap[normalized]; ok {
		return code
	}
	return normalized
}

// SymbolParts is a ticker split into its components.
type SymbolParts struct {
	Ticker   string // bare ticker, e.g. "IWDA"
	Exchange string // Yahoo suffix for stocks/ETFs; "" for US listings and crypto
	Currency string // quote currency for crypto; "" otherwise
}

// ParseSymbol splits a raw or stored symbol into its parts. It is the inverse of
// BuildSymbol for the two supported shapes (TICKER-CURRENCY, TICKER.EXCHANGE).
func ParseSymbol(raw string, investmentType models.InvestmentType) SymbolParts {
	value := strings.ToUpper(strings.TrimSpace(raw))

	if investmentType == models.InvestmentCrypto {
		if base, currency, ok := strings.Cut(value, "-"); ok {
			return SymbolParts{Ticker: base, Currency: currency}
		}
		return SymbolParts{Ticker: value}
	}

	if base, exchange, ok := strings.Cut(value, "."); ok {
		return SymbolParts{Ticker: base, Exchange: exchange}
	}
	return SymbolParts{Ticker: value}
}

// BuildSymbol composes the canonical Yahoo Finance symbol:
//   - crypto:    TICKER-CURRENCY  (currency defaults to USD)
//   - stock/ETF: TICKER.EXCHANGE  (or bare TICKER for US listings)
func BuildSymbol(parts SymbolParts, investmentType models.InvestmentType) string {
	ticker := strings.ToUpper(strings.TrimSpace(parts.Ticker))
	if ticker == "" {
		return ""
	}

	if investmentType == models.InvestmentCrypto {
		currency := strings.ToUpper(strings.TrimSpace(parts.Currency))
		if currency == "" {
			currency = "USD"
		}
		return ticker + "-" + currency
	}

	exchange := NormalizeExchange(parts.Exchange)
	if exchange == "" {
		return ticker
	}
	return ticker + "." + exchange
}

// NormalizeTicker cleans a user-entered ticker into its canonical Yahoo symbol.
// Crypto without a pair defaults to -USD; stock/ETF symbols must be a bare US
// ticker or TICKER.EXCHANGE with a known exchange suffix.
func NormalizeTicker(raw string, investmentType models.InvestmentType) (string, error) {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return "", fmt.Errorf("ticker cannot be empty")
	}

	if investmentType == models.InvestmentCrypto {
		return BuildSymbol(ParseSymbol(value, investmentType), investmentType), nil
	}

	if !stockSymbolRegex.MatchString(value) {
		return "", fmt.Errorf("invalid stock/ETF ticker: must look like AAPL or IWDA.AS")
	}
	return value, nil
}
