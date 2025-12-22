-- +goose Up
-- +goose StatementBegin
CREATE TYPE investment_type AS ENUM ('stock', 'etf', 'crypto');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE investment_holdings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,

    investment_type investment_type NOT NULL,
    name VARCHAR(255) NOT NULL,
    ticker VARCHAR(20) NOT NULL,
    quantity NUMERIC(19,8) NOT NULL,
    value_at_buy NUMERIC(19,4) NOT NULL,
    current_value NUMERIC(19,4) NOT NULL,
    average_buy_price NUMERIC(19,4) NOT NULL,
    profit_loss NUMERIC(19,4) NOT NULL,
    profit_loss_percent NUMERIC(19,4) NOT NULL,
    current_price NUMERIC(19,4),
    last_price_update TIMESTAMPTZ,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_holdings_account FOREIGN KEY (account_id)
        REFERENCES accounts(id),
    CONSTRAINT uq_account_ticker UNIQUE (account_id, ticker)
);

CREATE INDEX idx_holdings_account ON investment_holdings(account_id);
CREATE INDEX idx_holdings_user ON investment_holdings(user_id);
CREATE INDEX idx_holdings_ticker ON investment_holdings(ticker);

CREATE TRIGGER set_investment_holdings_updated_at
    BEFORE UPDATE ON investment_holdings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_investment_holdings_updated_at ON investment_holdings;
DROP TABLE IF EXISTS investment_holdings;
DROP TYPE IF EXISTS investment_type;
-- +goose StatementEnd
