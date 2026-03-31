-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN has_completed_setup BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE users
SET has_completed_setup = TRUE
WHERE id IN (SELECT user_id FROM settings_user);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
    DROP COLUMN has_completed_setup;
-- +goose StatementEnd
