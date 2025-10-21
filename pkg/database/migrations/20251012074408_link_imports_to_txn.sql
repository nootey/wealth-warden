-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN import_id BIGINT NULL;

ALTER TABLE transfers
    ADD COLUMN import_id BIGINT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions
    DROP COLUMN IF EXISTS import_id;

ALTER TABLE transfers
DROP COLUMN IF EXISTS import_id;
-- +goose StatementEnd