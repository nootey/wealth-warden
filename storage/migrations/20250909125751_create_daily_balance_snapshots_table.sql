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

CREATE OR REPLACE VIEW v_user_account_daily_snapshots AS
SELECT
    s.user_id,
    s.account_id,
    s.as_of,
    s.currency,
    s.end_balance::NUMERIC(19,4) AS end_balance
FROM account_daily_snapshots s
         JOIN accounts a
              ON a.id = s.account_id
WHERE
    a.include_in_net_worth = TRUE
    AND (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)   -- inclusive
    AND (a.closed_at IS NULL OR s.as_of::date <  a.closed_at::date);  -- exclusive

-- Aggregate per-user net worth from the filtered per-account view
CREATE OR REPLACE VIEW v_user_daily_networth_snapshots AS
SELECT
    s.user_id,
    s.as_of,
    s.currency,
    SUM(s.end_balance)::NUMERIC(19,4) AS end_balance
FROM v_user_account_daily_snapshots s
GROUP BY s.user_id, s.as_of, s.currency;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS v_user_daily_networth_snapshots;
DROP VIEW IF EXISTS v_user_account_daily_snapshots;
DROP INDEX IF EXISTS idx_ads_user_ccy_asof;
DROP INDEX IF EXISTS idx_ads_account_asof;
DROP INDEX IF EXISTS idx_ads_user_asof;
DROP TABLE IF EXISTS account_daily_snapshots;
-- +goose StatementEnd