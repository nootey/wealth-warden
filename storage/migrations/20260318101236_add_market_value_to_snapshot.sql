-- +goose Up
-- +goose StatementBegin
ALTER TABLE account_daily_snapshots
    ADD COLUMN market_value NUMERIC(19,4) NOT NULL DEFAULT 0;

-- Expose combined total (cash + market value) as end_balance so downstream queries
-- don't need to change — non-investment accounts always have market_value = 0
CREATE OR REPLACE VIEW v_user_account_daily_snapshots AS
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
  AND (a.closed_at IS NULL OR s.as_of::date <  a.closed_at::date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE VIEW v_user_account_daily_snapshots AS
SELECT
    s.user_id,
    s.account_id,
    s.as_of,
    s.currency,
    s.end_balance::NUMERIC(19,4) AS end_balance
FROM account_daily_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE
    a.include_in_net_worth = TRUE
  AND (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)
  AND (a.closed_at IS NULL OR s.as_of::date <  a.closed_at::date);

ALTER TABLE account_daily_snapshots
    DROP COLUMN market_value;
-- +goose StatementEnd
