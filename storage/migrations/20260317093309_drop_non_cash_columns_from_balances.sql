-- +goose Up
-- +goose StatementBegin
ALTER TABLE balances DROP COLUMN end_balance;
ALTER TABLE balances DROP COLUMN non_cash_inflows;
ALTER TABLE balances DROP COLUMN non_cash_outflows;
ALTER TABLE balances DROP COLUMN net_market_flows;
ALTER TABLE balances DROP COLUMN adjustments;
ALTER TABLE balances ADD COLUMN end_balance NUMERIC(19,4)
    GENERATED ALWAYS AS (
        start_balance
            + cash_inflows - cash_outflows
        ) STORED;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE balances DROP COLUMN end_balance;
ALTER TABLE balances ADD COLUMN non_cash_inflows  NUMERIC(19,4) NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN non_cash_outflows NUMERIC(19,4) NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN net_market_flows  NUMERIC(19,4) NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN adjustments       NUMERIC(19,4) NOT NULL DEFAULT 0;
ALTER TABLE balances ADD COLUMN end_balance NUMERIC(19,4)
    GENERATED ALWAYS AS (
        start_balance
            + cash_inflows - cash_outflows
            + non_cash_inflows - non_cash_outflows
            + net_market_flows
            + adjustments
        ) STORED;
-- +goose StatementEnd