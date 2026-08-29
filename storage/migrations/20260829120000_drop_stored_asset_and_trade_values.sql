-- +goose Up
-- +goose StatementBegin
ALTER TABLE investment_assets
    DROP COLUMN current_value,
    DROP COLUMN profit_loss,
    DROP COLUMN profit_loss_percent,
    DROP COLUMN last_price_update;

ALTER TABLE investment_trades
    DROP COLUMN current_value,
    DROP COLUMN profit_loss,
    DROP COLUMN profit_loss_percent;

CREATE VIEW ticker_latest_price AS
SELECT DISTINCT ON (ticker) ticker, price, currency, updated_at
FROM   ticker_price_history
ORDER  BY ticker, as_of DESC;

CREATE VIEW investment_assets_valued AS
SELECT ia.*,
       lp.updated_at AS last_price_update,
       lp.price AS latest_price,
       lp.currency AS latest_price_currency,
       COALESCE(lp.price * ia.quantity, 0) AS current_value,
       CASE WHEN lp.price IS NULL THEN 0
            ELSE lp.price * ia.quantity
                 - (ia.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE ia.total_fees END)
       END AS profit_loss,
       CASE WHEN lp.price IS NULL
             OR (ia.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE ia.total_fees END) <= 0
            THEN 0
            ELSE (lp.price * ia.quantity
                  - (ia.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE ia.total_fees END))
                 / (ia.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE ia.total_fees END)
       END AS profit_loss_percent
FROM   investment_assets ia
LEFT   JOIN ticker_latest_price lp ON lp.ticker = ia.ticker;

CREATE VIEW investment_trades_valued AS
SELECT it.*,
       CASE WHEN it.trade_type = 'buy'
            THEN COALESCE(lp.price * it.quantity, 0)
            ELSE it.quantity * it.price_per_unit
       END AS current_value,
       CASE WHEN it.trade_type = 'buy'
            THEN CASE WHEN lp.price IS NULL THEN 0
                      ELSE lp.price * it.quantity
                           - (it.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE it.fee END)
                 END
            ELSE it.realized_value - it.value_at_buy
       END AS profit_loss,
       CASE WHEN it.trade_type = 'buy'
            THEN CASE WHEN lp.price IS NULL
                       OR (it.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE it.fee END) <= 0
                      THEN 0
                      ELSE (lp.price * it.quantity
                            - (it.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE it.fee END))
                           / (it.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE it.fee END)
                 END
            ELSE CASE WHEN it.value_at_buy <= 0 THEN 0
                      ELSE (it.realized_value - it.value_at_buy) / it.value_at_buy
                 END
       END AS profit_loss_percent
FROM   investment_trades it
JOIN   investment_assets ia ON ia.id = it.asset_id
LEFT   JOIN ticker_latest_price lp ON lp.ticker = ia.ticker;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS investment_trades_valued;
DROP VIEW IF EXISTS investment_assets_valued;
DROP VIEW IF EXISTS ticker_latest_price;

ALTER TABLE investment_assets
    ADD COLUMN current_value       NUMERIC(19,4) NOT NULL DEFAULT 0,
    ADD COLUMN profit_loss         NUMERIC(19,4) NOT NULL DEFAULT 0,
    ADD COLUMN profit_loss_percent NUMERIC(10,2) NOT NULL DEFAULT 0,
    ADD COLUMN last_price_update   TIMESTAMPTZ;

ALTER TABLE investment_trades
    ADD COLUMN current_value       NUMERIC(19,4) NOT NULL DEFAULT 0,
    ADD COLUMN profit_loss         NUMERIC(19,4) NOT NULL DEFAULT 0,
    ADD COLUMN profit_loss_percent NUMERIC(10,2) NOT NULL DEFAULT 0;

UPDATE investment_assets ia
SET    current_value = COALESCE(lp.price, 0) * ia.quantity,
       profit_loss = CASE WHEN lp.price IS NULL THEN 0
           ELSE lp.price * ia.quantity
                - (ia.value_at_buy + CASE WHEN ia.investment_type = 'crypto' THEN 0 ELSE ia.total_fees END)
       END,
       last_price_update = lp.updated_at
FROM (
    SELECT DISTINCT ON (ticker) ticker, price, updated_at
    FROM   ticker_price_history
    ORDER  BY ticker, as_of DESC
) lp
WHERE lp.ticker = ia.ticker;
-- +goose StatementEnd
