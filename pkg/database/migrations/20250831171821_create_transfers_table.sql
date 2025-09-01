-- +goose Up
-- +goose StatementBegin
CREATE TYPE transfer_status_enum AS ENUM ('pending', 'success', 'failed', 'rejected');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transfers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    transaction_inflow_id BIGINT NOT NULL,
    transaction_outflow_id BIGINT NOT NULL,
    status transfer_status_enum NOT NULL DEFAULT 'pending',
    amount NUMERIC(19,4) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_transactions_user     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_transactions_inflow  FOREIGN KEY (transaction_inflow_id) REFERENCES transactions(id),
    CONSTRAINT fk_transactions_outflow FOREIGN KEY (transaction_outflow_id) REFERENCES transactions(id),
    CONSTRAINT uq_transaction UNIQUE (transaction_inflow_id, transaction_outflow_id)
);

CREATE INDEX idx_transfer_transaction_inflow ON transfers(transaction_inflow_id);
CREATE INDEX idx_transfer_transaction_outflow ON transfers(transaction_outflow_id);
CREATE INDEX idx_transfer_status ON transfers(status);

CREATE TRIGGER set_transfers_updated_at
    BEFORE UPDATE ON transfers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_transfers_updated_at ON transfers;
DROP TABLE IF EXISTS transfers;
-- +goose StatementEnd
