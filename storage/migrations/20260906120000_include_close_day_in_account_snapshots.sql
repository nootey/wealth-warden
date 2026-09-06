-- +goose Up
-- +goose StatementBegin
-- CloseAccount writes a final snapshot for the close day, but the old `<` bound
-- filtered it out, so the account left the charts a day early.
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
  AND (a.closed_at IS NULL OR s.as_of::date <= a.closed_at::date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
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
