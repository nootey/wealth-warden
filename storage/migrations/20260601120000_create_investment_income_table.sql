-- +goose Up
-- +goose StatementBegin
CREATE TYPE income_type AS ENUM ('staking_reward', 'dividend');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE investment_income (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    asset_id BIGINT NOT NULL,

    txn_date DATE NOT NULL,
    income_type income_type NOT NULL,
    quantity NUMERIC(19,8),
    amount NUMERIC(19,4) NOT NULL,
    tax_withheld NUMERIC(19,4),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    notes VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_income_asset FOREIGN KEY (asset_id)
        REFERENCES investment_assets(id) ON DELETE CASCADE
);

CREATE INDEX idx_inv_income_asset ON investment_income(asset_id);
CREATE INDEX idx_inv_income_user ON investment_income(user_id);
CREATE INDEX idx_inv_income_date ON investment_income(txn_date);

CREATE TRIGGER set_investment_income_updated_at
    BEFORE UPDATE ON investment_income
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_investment_income_updated_at ON investment_income;
DROP TABLE IF EXISTS investment_income;
DROP TYPE IF EXISTS income_type;
-- +goose StatementEnd
