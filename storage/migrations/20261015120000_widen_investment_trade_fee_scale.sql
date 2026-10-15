-- +goose Up
-- +goose StatementBegin
-- Postgres refuses to retype a column a view reads, so the view is rebuilt around the change.
DROP VIEW IF EXISTS investment_trades_valued;

ALTER TABLE investment_trades ALTER COLUMN fee TYPE NUMERIC(19,6);

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

ALTER TABLE investment_trades ALTER COLUMN fee TYPE NUMERIC(19,4);

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
