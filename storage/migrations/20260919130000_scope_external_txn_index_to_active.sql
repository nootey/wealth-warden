-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_external_txn;

CREATE UNIQUE INDEX idx_transactions_external_txn
    ON transactions (account_id, external_txn_id)
    WHERE external_txn_id IS NOT NULL AND deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_external_txn;

CREATE UNIQUE INDEX idx_transactions_external_txn
    ON transactions (account_id, external_txn_id)
    WHERE external_txn_id IS NOT NULL;
-- +goose StatementEnd
