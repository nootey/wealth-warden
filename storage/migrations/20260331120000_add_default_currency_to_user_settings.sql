-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings_user
    ADD COLUMN default_currency TEXT NOT NULL DEFAULT 'EUR';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings_user
    DROP COLUMN default_currency;
-- +goose StatementEnd
