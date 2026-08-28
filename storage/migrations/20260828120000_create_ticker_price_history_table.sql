-- +goose Up
-- +goose StatementBegin
CREATE TABLE ticker_price_history (
    ticker     VARCHAR(20)   NOT NULL,
    as_of      DATE          NOT NULL,
    price      NUMERIC(19,4) NOT NULL,
    currency   CHAR(3)       NOT NULL DEFAULT 'USD',

    updated_at TIMESTAMPTZ   DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (ticker, as_of)
);

CREATE INDEX idx_tph_ticker_asof ON ticker_price_history(ticker, as_of);

CREATE TRIGGER set_ticker_price_history_updated_at
    BEFORE UPDATE ON ticker_price_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

INSERT INTO ticker_price_history (ticker, as_of, price, currency, created_at, updated_at)
SELECT DISTINCT ON (ia.ticker, aph.as_of)
       ia.ticker, aph.as_of, aph.price, aph.currency, aph.created_at, aph.updated_at
FROM   asset_price_history aph
JOIN   investment_assets ia ON ia.id = aph.asset_id
ORDER  BY ia.ticker, aph.as_of, aph.updated_at DESC NULLS LAST, aph.created_at DESC;

DROP TABLE asset_price_history;

ALTER TABLE investment_assets DROP COLUMN current_price;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE investment_assets ADD COLUMN current_price NUMERIC(19,4);

CREATE TABLE asset_price_history (
    asset_id   BIGINT        NOT NULL,
    as_of      DATE          NOT NULL,
    price      NUMERIC(19,4) NOT NULL,
    currency   CHAR(3)       NOT NULL DEFAULT 'USD',

    updated_at TIMESTAMPTZ   DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (asset_id, as_of),
    CONSTRAINT fk_aph_asset FOREIGN KEY (asset_id)
        REFERENCES investment_assets(id) ON DELETE CASCADE
);

CREATE INDEX idx_aph_asset_asof ON asset_price_history(asset_id, as_of);

CREATE TRIGGER set_asset_price_history_updated_at
    BEFORE UPDATE ON asset_price_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

INSERT INTO asset_price_history (asset_id, as_of, price, currency, created_at, updated_at)
SELECT ia.id, tph.as_of, tph.price, tph.currency, tph.created_at, tph.updated_at
FROM   ticker_price_history tph
JOIN   investment_assets ia ON ia.ticker = tph.ticker;

UPDATE investment_assets ia
SET    current_price = latest.price
FROM   (
    SELECT DISTINCT ON (ticker) ticker, price
    FROM   ticker_price_history
    ORDER  BY ticker, as_of DESC
) latest
WHERE  latest.ticker = ia.ticker;

DROP TABLE ticker_price_history;
-- +goose StatementEnd
