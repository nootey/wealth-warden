-- +goose Up
-- +goose StatementBegin
-- Pins the rate to USD on txn_date, the same way investment_trades does, so a past
-- dividend keeps its rate if the provider revises history. Existing rows default
-- to 1 and are corrected by the backfill_income_fx_rates job.
ALTER TABLE investment_income
    ADD COLUMN exchange_rate_to_usd NUMERIC(19,6) NOT NULL DEFAULT 1.0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE investment_income
    DROP COLUMN exchange_rate_to_usd;
-- +goose StatementEnd
