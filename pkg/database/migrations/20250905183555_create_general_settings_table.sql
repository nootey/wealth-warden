-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings_general (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    support_email TEXT,
    allow_signups BOOLEAN NOT NULL DEFAULT TRUE,
    default_locale   TEXT NOT NULL DEFAULT 'en',
    default_timezone TEXT NOT NULL DEFAULT 'UTC',
    max_accounts_per_user SMALLINT NOT NULL DEFAULT 25,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS settings_general_one_row
    ON settings_general ((true));

CREATE TRIGGER set_settings_general_updated_at
    BEFORE UPDATE ON settings_general
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_settings_general_updated_at ON settings_general;
DROP TABLE IF EXISTS settings_general;
-- +goose StatementEnd
