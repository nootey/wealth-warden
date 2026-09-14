-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN external_txn_id VARCHAR(256) NULL;

CREATE UNIQUE INDEX idx_transactions_external_txn
    ON transactions (account_id, external_txn_id)
    WHERE external_txn_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_external_txn;
ALTER TABLE transactions DROP COLUMN IF EXISTS external_txn_id;
-- +goose StatementEnd
