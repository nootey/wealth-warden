-- +goose Up
-- +goose StatementBegin
ALTER TABLE settings_user
    ADD COLUMN default_sheet_separator TEXT NOT NULL DEFAULT ';';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE settings_user
    DROP COLUMN default_sheet_separator;
-- +goose StatementEnd
