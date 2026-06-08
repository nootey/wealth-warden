-- +goose Up
-- +goose StatementBegin
CREATE TABLE investment_tax_settings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,

    loss_offsetting_enabled BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_tax_settings_user UNIQUE (user_id)
);

CREATE TRIGGER set_investment_tax_settings_updated_at
    BEFORE UPDATE ON investment_tax_settings
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_investment_tax_settings_updated_at ON investment_tax_settings;
DROP TABLE IF EXISTS investment_tax_settings;
-- +goose StatementEnd
