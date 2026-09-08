-- +goose Up
-- +goose StatementBegin
-- Renames only. The daily table no longer sits beside a per-day `balances`
-- chain, so "account_daily" stopped telling the reader anything.
ALTER TABLE account_daily_snapshots RENAME TO balance_snapshots;

ALTER INDEX account_daily_snapshots_pkey RENAME TO balance_snapshots_pkey;
ALTER INDEX idx_ads_user_asof     RENAME TO idx_bs_user_asof;
ALTER INDEX idx_ads_account_asof  RENAME TO idx_bs_account_asof;
ALTER INDEX idx_ads_user_ccy_asof RENAME TO idx_bs_user_ccy_asof;

ALTER TABLE balance_snapshots RENAME CONSTRAINT fk_ads_user         TO fk_bs_user;
ALTER TABLE balance_snapshots RENAME CONSTRAINT fk_ads_account      TO fk_bs_account;
ALTER TABLE balance_snapshots RENAME CONSTRAINT uq_ads_account_asof TO uq_bs_account_asof;
ALTER TABLE balance_snapshots RENAME CONSTRAINT chk_ads_currency    TO chk_bs_currency;

-- A view cannot be renamed by CREATE OR REPLACE, and the networth view depends
-- on the account view, so both are dropped and rebuilt in order.
DROP VIEW IF EXISTS v_user_daily_networth_snapshots;
DROP VIEW IF EXISTS v_user_account_daily_snapshots;

CREATE VIEW v_user_account_balance_snapshots AS
SELECT
    s.user_id,
    s.account_id,
    s.as_of,
    s.currency,
    (s.end_balance + s.market_value)::NUMERIC(19,4) AS end_balance
FROM balance_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE
    a.include_in_net_worth = TRUE
  AND (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)
  AND (a.closed_at IS NULL OR s.as_of::date <= a.closed_at::date);

CREATE VIEW v_user_networth_snapshots AS
SELECT
    s.user_id,
    s.as_of,
    s.currency,
    SUM(s.end_balance)::NUMERIC(19,4) AS end_balance
FROM v_user_account_balance_snapshots s
GROUP BY s.user_id, s.as_of, s.currency;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS v_user_networth_snapshots;
DROP VIEW IF EXISTS v_user_account_balance_snapshots;

ALTER TABLE balance_snapshots RENAME CONSTRAINT chk_bs_currency    TO chk_ads_currency;
ALTER TABLE balance_snapshots RENAME CONSTRAINT uq_bs_account_asof TO uq_ads_account_asof;
ALTER TABLE balance_snapshots RENAME CONSTRAINT fk_bs_account      TO fk_ads_account;
ALTER TABLE balance_snapshots RENAME CONSTRAINT fk_bs_user         TO fk_ads_user;

ALTER INDEX idx_bs_user_ccy_asof RENAME TO idx_ads_user_ccy_asof;
ALTER INDEX idx_bs_account_asof  RENAME TO idx_ads_account_asof;
ALTER INDEX idx_bs_user_asof     RENAME TO idx_ads_user_asof;
ALTER INDEX balance_snapshots_pkey RENAME TO account_daily_snapshots_pkey;

ALTER TABLE balance_snapshots RENAME TO account_daily_snapshots;

CREATE VIEW v_user_account_daily_snapshots AS
SELECT
    s.user_id,
    s.account_id,
    s.as_of,
    s.currency,
    (s.end_balance + s.market_value)::NUMERIC(19,4) AS end_balance
FROM account_daily_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE
    a.include_in_net_worth = TRUE
  AND (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)
  AND (a.closed_at IS NULL OR s.as_of::date <= a.closed_at::date);

CREATE VIEW v_user_daily_networth_snapshots AS
SELECT
    s.user_id,
    s.as_of,
    s.currency,
    SUM(s.end_balance)::NUMERIC(19,4) AS end_balance
FROM v_user_account_daily_snapshots s
GROUP BY s.user_id, s.as_of, s.currency;
-- +goose StatementEnd
