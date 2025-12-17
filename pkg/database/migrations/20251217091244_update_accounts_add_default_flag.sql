-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts
    ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;

-- Ensure only one default account per (user_id, account_type_id) combination
CREATE UNIQUE INDEX idx_one_default_per_user_account_type
    ON accounts(user_id, account_type_id)
    WHERE is_default = TRUE AND closed_at IS NULL AND is_active = TRUE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_one_default_per_user_account_type;
ALTER TABLE accounts DROP COLUMN IF EXISTS is_default;
-- +goose StatementEnd