-- +goose Up
-- +goose StatementBegin
-- An account left out of net worth still needs its own balance chart, so the
-- include_in_net_worth filter moves from the account view to the networth view.
CREATE OR REPLACE VIEW v_user_account_balance_snapshots AS
SELECT
    s.user_id,
    s.account_id,
    s.as_of,
    s.currency,
    (s.end_balance + s.market_value)::NUMERIC(19,4) AS end_balance
FROM balance_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE
    (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)
  AND (a.closed_at IS NULL OR s.as_of::date <= a.closed_at::date);

CREATE OR REPLACE VIEW v_user_networth_snapshots AS
SELECT
    s.user_id,
    s.as_of,
    s.currency,
    SUM(s.end_balance)::NUMERIC(19,4) AS end_balance
FROM v_user_account_balance_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE a.include_in_net_worth = TRUE
GROUP BY s.user_id, s.as_of, s.currency;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE VIEW v_user_networth_snapshots AS
SELECT
    s.user_id,
    s.as_of,
    s.currency,
    SUM(s.end_balance)::NUMERIC(19,4) AS end_balance
FROM v_user_account_balance_snapshots s
GROUP BY s.user_id, s.as_of, s.currency;

CREATE OR REPLACE VIEW v_user_account_balance_snapshots AS
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
-- +goose StatementEnd
