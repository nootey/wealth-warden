-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE VIEW v_user_account_daily_snapshots AS
SELECT
    s.user_id, s.account_id, s.as_of, s.currency,
    s.end_balance::NUMERIC(19,4) AS end_balance
FROM account_daily_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE
    a.include_in_net_worth = TRUE
  AND (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)
  AND (a.closed_at IS NULL OR s.as_of::date <  a.closed_at::date);

CREATE OR REPLACE VIEW v_user_daily_networth_snapshots AS
WITH holdings AS (
    SELECT
        ia.user_id,
        ia.currency,
        ph.as_of,
        SUM(
                ph.price * (
                    SELECT COALESCE(SUM(
                                            CASE WHEN it.trade_type = 'buy'  THEN it.quantity
                                                 WHEN it.trade_type = 'sell' THEN -it.quantity
                                                END
                                    ), 0)
                    FROM investment_trades it
                    WHERE it.asset_id = ia.id
                      AND it.txn_date <= ph.as_of
                )
        ) AS market_value
    FROM asset_price_history ph
             JOIN investment_assets ia ON ia.id = ph.asset_id
    GROUP BY ia.user_id, ia.currency, ph.as_of
)
SELECT
    s.user_id,
    s.as_of,
    s.currency,
    (SUM(s.end_balance) + COALESCE(SUM(h.market_value), 0))::NUMERIC(19,4) AS end_balance
FROM v_user_account_daily_snapshots s
         LEFT JOIN holdings h
                   ON  h.user_id  = s.user_id
                       AND h.as_of    = s.as_of
                       AND h.currency = s.currency
GROUP BY s.user_id, s.as_of, s.currency;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE VIEW v_user_account_daily_snapshots AS
SELECT
    s.user_id, s.account_id, s.as_of, s.currency,
    s.end_balance::NUMERIC(19,4) AS end_balance
FROM account_daily_snapshots s
         JOIN accounts a ON a.id = s.account_id
WHERE
    a.include_in_net_worth = TRUE
  AND (a.opened_at IS NULL OR s.as_of::date >= a.opened_at::date)
  AND (a.closed_at IS NULL OR s.as_of::date <  a.closed_at::date);

CREATE OR REPLACE VIEW v_user_daily_networth_snapshots AS
SELECT user_id, as_of, currency, SUM(end_balance)::NUMERIC(19,4) AS end_balance
FROM v_user_account_daily_snapshots
GROUP BY user_id, as_of, currency;
-- +goose StatementEnd