-- +goose Up
-- +goose StatementBegin
CREATE TABLE balances (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL,
    as_of DATE NOT NULL,

    start_balance     NUMERIC(19,4) NOT NULL DEFAULT 0,
    cash_inflows      NUMERIC(19,4) NOT NULL DEFAULT 0,
    cash_outflows     NUMERIC(19,4) NOT NULL DEFAULT 0,
    non_cash_inflows  NUMERIC(19,4) NOT NULL DEFAULT 0,
    non_cash_outflows NUMERIC(19,4) NOT NULL DEFAULT 0,
    net_market_flows  NUMERIC(19,4) NOT NULL DEFAULT 0,
    adjustments       NUMERIC(19,4) NOT NULL DEFAULT 0,

    end_balance NUMERIC(19,4) GENERATED ALWAYS AS (
                                     start_balance
                                     + cash_inflows - cash_outflows
                             + non_cash_inflows - non_cash_outflows
                     + net_market_flows
                     + adjustments
                 ) STORED,

    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_balances_account FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    CONSTRAINT uq_account_asof UNIQUE (account_id, as_of)
);

CREATE INDEX idx_balances_account_asof ON balances(account_id, as_of);

CREATE TRIGGER set_balances_updated_at
    BEFORE UPDATE ON balances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_balances_updated_at ON balances;
DROP TABLE IF EXISTS balances;
-- +goose StatementEnd
