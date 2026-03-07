-- +goose Up
-- +goose StatementBegin
ALTER TABLE transaction_templates ADD COLUMN day_of_month INT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transaction_templates DROP COLUMN day_of_month;
-- +goose StatementEnd