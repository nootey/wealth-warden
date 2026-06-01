-- +goose Up
ALTER TABLE transactions ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE investment_income ADD COLUMN linked_transaction_id BIGINT REFERENCES transactions(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE investment_income DROP COLUMN linked_transaction_id;
ALTER TABLE transactions DROP COLUMN is_system;
