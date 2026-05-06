-- +goose Up
-- +goose StatementBegin
-- total_fees accumulates all trade fees for this asset position.
-- For crypto assets: fees are denominated in the crypto token (e.g. 0.001 BTC).
--   They are already deducted from the effective quantity and are recorded here
--   for informational purposes only - they are not used in P&L calculations.
-- For stocks/ETFs: fees are denominated in the trade currency (e.g. EUR, USD).
--   They represent broker commissions and should be added to value_at_buy
--   when calculating true cost basis and P&L.
ALTER TABLE investment_assets
    ADD COLUMN total_fees NUMERIC(19,6) NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE investment_assets
    DROP COLUMN total_fees;
-- +goose StatementEnd
