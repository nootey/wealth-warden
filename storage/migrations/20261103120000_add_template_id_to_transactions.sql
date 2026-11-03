-- +goose Up
-- +goose StatementBegin
-- No foreign key: ON DELETE SET NULL would update soft-deleted rows, and a trigger blocks that.
ALTER TABLE transactions
    ADD COLUMN template_id BIGINT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP COLUMN IF EXISTS template_id;
-- +goose StatementEnd
