-- +goose Up
-- +goose StatementBegin
CREATE TABLE investment_tax_brackets (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,

    investment_type investment_type NOT NULL,
    min_days_held INT NOT NULL,
    to_days INT,
    taxable_percent NUMERIC(5,2) NOT NULL,
    label VARCHAR(100),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_taxable_percent CHECK (taxable_percent >= 0 AND taxable_percent <= 100),
    CONSTRAINT chk_min_days_held CHECK (min_days_held >= 0),
    CONSTRAINT uq_tax_bracket UNIQUE (user_id, investment_type, min_days_held)
);

CREATE INDEX idx_tax_brackets_user ON investment_tax_brackets(user_id);

CREATE TRIGGER set_investment_tax_brackets_updated_at
    BEFORE UPDATE ON investment_tax_brackets
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_investment_tax_brackets_updated_at ON investment_tax_brackets;
DROP TABLE IF EXISTS investment_tax_brackets;
-- +goose StatementEnd
