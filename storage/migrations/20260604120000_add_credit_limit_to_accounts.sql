-- +goose Up
-- +goose StatementBegin
ALTER TABLE accounts ADD COLUMN credit_limit NUMERIC(19,4) NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE accounts DROP COLUMN IF EXISTS credit_limit;
-- +goose StatementEnd
