-- +goose Up
-- +goose StatementBegin
ALTER TYPE transaction_type_enum RENAME TO transaction_direction_enum;
ALTER TABLE transactions RENAME COLUMN transaction_type TO direction;
ALTER TABLE transaction_templates RENAME COLUMN transaction_type TO direction;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transaction_templates RENAME COLUMN direction TO transaction_type;
ALTER TABLE transactions RENAME COLUMN direction TO transaction_type;
ALTER TYPE transaction_direction_enum RENAME TO transaction_type_enum;
-- +goose StatementEnd
