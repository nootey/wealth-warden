-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tokens (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token_type VARCHAR(50) NOT NULL, -- email_verify, password_reset ...
    token_value TEXT NOT NULL, -- value of the stored token
    data JSONB, -- additional singular piece of data, relating to the token (optional)
    CONSTRAINT tokens_token_type_data_unique UNIQUE (token_type, data),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER set_tokens_updated_at
    BEFORE UPDATE ON tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tokens CASCADE;
-- +goose StatementEnd