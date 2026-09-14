-- +goose Up
-- +goose StatementBegin
ALTER TABLE rules
    ADD COLUMN import_id BIGINT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE rules
    DROP COLUMN IF EXISTS import_id;
-- +goose StatementEnd
