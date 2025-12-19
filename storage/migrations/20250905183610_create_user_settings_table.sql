-- +goose Up
-- +goose StatementBegin
CREATE TABLE settings_user (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    theme     TEXT NOT NULL DEFAULT 'system' CHECK (theme IN ('light','dark','system')),
    accent TEXT,
    language  TEXT,
    timezone  TEXT,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);


CREATE TRIGGER set_settings_user_updated_at
    BEFORE UPDATE ON settings_user
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_settings_user_updated_at ON settings_user;
DROP TABLE IF EXISTS settings_user;
-- +goose StatementEnd
