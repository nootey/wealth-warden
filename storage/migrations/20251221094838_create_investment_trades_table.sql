-- +goose Up
-- +goose StatementBegin
CREATE TABLE investment_trades (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    asset_id BIGINT NOT NULL,

    txn_date DATE NOT NULL,
    trade_type VARCHAR(4) NOT NULL CHECK (trade_type IN ('buy', 'sell')),
    quantity NUMERIC(19,8) NOT NULL,
    fee NUMERIC(19,4) NOT NULL DEFAULT 0,
    price_per_unit NUMERIC(19,4) NOT NULL,
    value_at_buy NUMERIC(19,4) NOT NULL,
    current_value NUMERIC(19,4) NOT NULL,
    realized_value NUMERIC(19,4) NOT NULL default 0,
    profit_loss NUMERIC(19,4) NOT NULL,
    profit_loss_percent NUMERIC(19,4) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    exchange_rate_to_usd NUMERIC(19,6) NOT NULL DEFAULT 1.0,
    description VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_inv_trans_asset FOREIGN KEY (asset_id)
        REFERENCES investment_assets(id)
);

CREATE INDEX idx_inv_trans_asset ON investment_trades(asset_id);
CREATE INDEX idx_inv_trans_date ON investment_trades(txn_date);

CREATE TRIGGER set_investment_trades_updated_at
    BEFORE UPDATE ON investment_trades
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_investment_trades_updated_at ON investment_trades;
DROP TABLE IF EXISTS investment_trades;
-- +goose StatementEnd
