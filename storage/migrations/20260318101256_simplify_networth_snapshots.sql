-- +goose Up
-- +goose StatementBegin
-- market_value is now stored directly in account_daily_snapshots and included
-- in v_user_account_daily_snapshots.end_balance, so net worth is a plain sum.
-- No more complex holdings CTE or exact-date joins against asset_price_history.
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
-- Restore the holdings-CTE version that was set by update_views_to_include_asset_market_value
CREATE OR REPLACE VIEW v_user_daily_networth_snapshots AS
WITH holdings AS (
    SELECT
        ia.user_id,
        ia.currency,
        ph.as_of,
        SUM(
            ph.price * (
                SELECT COALESCE(SUM(
                    CASE WHEN it.trade_type = 'buy'  THEN  it.quantity
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
