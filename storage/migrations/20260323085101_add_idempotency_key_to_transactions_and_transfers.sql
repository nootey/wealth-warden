-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN idempotency_key VARCHAR(64) NULL;

CREATE UNIQUE INDEX idx_transactions_idempotency
    ON transactions (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

ALTER TABLE transfers
    ADD COLUMN idempotency_key VARCHAR(64) NULL;

CREATE UNIQUE INDEX idx_transfers_idempotency
    ON transfers (user_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_idempotency;
ALTER TABLE transactions DROP COLUMN IF EXISTS idempotency_key;

DROP INDEX IF EXISTS idx_transfers_idempotency;
ALTER TABLE transfers DROP COLUMN IF EXISTS idempotency_key;
-- +goose StatementEnd
