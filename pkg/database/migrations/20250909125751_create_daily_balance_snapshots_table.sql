-- +goose Up
-- +goose StatementBegin
CREATE TABLE account_daily_snapshots (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    as_of DATE NOT NULL,
    end_balance NUMERIC(19,4) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    computed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_ads_user    FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_ads_account FOREIGN KEY (account_id) REFERENCES accounts(id),
    CONSTRAINT uq_ads_account_asof UNIQUE (account_id, as_of),
    CONSTRAINT chk_ads_currency CHECK (currency ~ '^[A-Z]{3}$')
);

-- Reads by user & range
CREATE INDEX idx_ads_user_asof     ON account_daily_snapshots(user_id, as_of);
-- Reads by account & range (writer/backfill)
CREATE INDEX idx_ads_account_asof  ON account_daily_snapshots(account_id, as_of);
-- Fast per-currency charts (common in multi-currency)
CREATE INDEX idx_ads_user_ccy_asof ON account_daily_snapshots(user_id, currency, as_of);

-- Per-user daily net worth from snapshots (currency-separated)
CREATE OR REPLACE VIEW v_user_daily_networth_snapshots AS
SELECT
    user_id,
    as_of,
    currency,
    SUM(end_balance)::NUMERIC(19,4) AS end_balance
FROM account_daily_snapshots
GROUP BY user_id, as_of, currency;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS v_user_daily_networth_snapshots;
DROP INDEX IF EXISTS idx_ads_user_ccy_asof;
DROP INDEX IF EXISTS idx_ads_account_asof;
DROP INDEX IF EXISTS idx_ads_user_asof;
DROP TABLE IF EXISTS account_daily_snapshots;
-- +goose StatementEnd