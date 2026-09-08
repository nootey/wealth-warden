-- +goose Up
-- +goose StatementBegin
-- The per-day chain is gone. transactions is the audit trail, account_daily_snapshots
-- carries the history, and the one row table is the number the app reads.
DROP TRIGGER IF EXISTS set_balances_updated_at ON balances;
DROP INDEX IF EXISTS idx_balances_account_asof;
DROP TABLE IF EXISTS balances;

ALTER TABLE account_balances RENAME TO balances;
ALTER INDEX account_balances_pkey RENAME TO balances_pkey;
ALTER INDEX idx_account_balances_user RENAME TO idx_balances_user;
ALTER TABLE balances RENAME CONSTRAINT fk_account_balances_account TO fk_balances_account;
ALTER TRIGGER set_account_balances_updated_at ON balances RENAME TO set_balances_updated_at;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- The old rows cannot come back. This restores the shape only.
ALTER TRIGGER set_balances_updated_at ON balances RENAME TO set_account_balances_updated_at;
ALTER TABLE balances RENAME CONSTRAINT fk_balances_account TO fk_account_balances_account;
ALTER INDEX idx_balances_user RENAME TO idx_account_balances_user;
ALTER INDEX balances_pkey RENAME TO account_balances_pkey;
ALTER TABLE balances RENAME TO account_balances;

CREATE TABLE balances (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT NOT NULL,
    as_of DATE NOT NULL,

    start_balance NUMERIC(19,4) NOT NULL DEFAULT 0,
    cash_inflows  NUMERIC(19,4) NOT NULL DEFAULT 0,
    cash_outflows NUMERIC(19,4) NOT NULL DEFAULT 0,

    end_balance NUMERIC(19,4) GENERATED ALWAYS AS (
        start_balance + cash_inflows - cash_outflows
    ) STORED,

    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_balances_account FOREIGN KEY (account_id) REFERENCES accounts(id),
    CONSTRAINT uq_account_asof UNIQUE (account_id, as_of)
);

CREATE INDEX idx_balances_account_asof ON balances(account_id, as_of);

CREATE TRIGGER set_balances_updated_at
    BEFORE UPDATE ON balances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd
