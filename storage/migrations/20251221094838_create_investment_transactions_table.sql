-- +goose Up
-- +goose StatementBegin
CREATE TABLE investment_transactions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    holding_id BIGINT NOT NULL,

    txn_date DATE NOT NULL,
    transaction_type VARCHAR(4) NOT NULL CHECK (transaction_type IN ('buy', 'sell')),
    quantity NUMERIC(19,8) NOT NULL,
    fee NUMERIC(19,4) NOT NULL DEFAULT 0,
    price_per_unit NUMERIC(19,4) NOT NULL,
    value_at_buy NUMERIC(19,4) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    exchange_rate_to_usd NUMERIC(19,6) NOT NULL DEFAULT 1.0,
    description VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_inv_trans_holding FOREIGN KEY (holding_id)
        REFERENCES investment_holdings(id)
);

CREATE INDEX idx_inv_trans_holding ON investment_transactions(holding_id);
CREATE INDEX idx_inv_trans_date ON investment_transactions(txn_date);

CREATE TRIGGER set_investment_transactions_updated_at
    BEFORE UPDATE ON investment_transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_investment_transactions_updated_at ON investment_transactions;
DROP TABLE IF EXISTS investment_transactions;
-- +goose StatementEnd
