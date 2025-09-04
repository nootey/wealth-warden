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
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_transactions_user     FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_transactions_inflow  FOREIGN KEY (transaction_inflow_id) REFERENCES transactions(id),
    CONSTRAINT fk_transactions_outflow FOREIGN KEY (transaction_outflow_id) REFERENCES transactions(id)
);

-- each tx can belong to at most one active transfer
CREATE UNIQUE INDEX uq_transfer_inflow_active
    ON transfers(transaction_inflow_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_transfer_outflow_active
    ON transfers(transaction_outflow_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uq_transfer_pair_active
    ON transfers(transaction_inflow_id, transaction_outflow_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_transfer_inflow_active
    ON transfers(transaction_inflow_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_transfer_outflow_active
    ON transfers(transaction_outflow_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_transfer_status_active
    ON transfers(status)
    WHERE deleted_at IS NULL;

-- Block updates if soft-deleted
CREATE OR REPLACE FUNCTION prevent_update_of_soft_deleted_transfer()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF OLD.deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'Transfer % is soft-deleted; updates are not allowed', OLD.id
            USING ERRCODE = 'read_only_sql_transaction';
END IF;
RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_transfer_block_update_if_deleted ON transfers;
CREATE TRIGGER trg_transfer_block_update_if_deleted
    BEFORE UPDATE ON transfers
    FOR EACH ROW
    EXECUTE FUNCTION prevent_update_of_soft_deleted_transfer();

-- Soft delete
CREATE OR REPLACE FUNCTION soft_delete_transfer()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
UPDATE transfers
SET deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP),
    updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_soft_delete_transfers ON transfers;
CREATE TRIGGER trg_soft_delete_transfers
    BEFORE DELETE ON transfers
    FOR EACH ROW
    EXECUTE FUNCTION soft_delete_transfer();

CREATE TRIGGER set_transfers_updated_at
    BEFORE UPDATE ON transfers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_transfers_updated_at ON transfers;
DROP TRIGGER IF EXISTS trg_soft_delete_transfers ON transfers;
DROP FUNCTION IF EXISTS soft_delete_transfer();
DROP TRIGGER IF EXISTS trg_transfer_block_update_if_deleted ON transfers;
DROP FUNCTION IF EXISTS prevent_update_of_soft_deleted_transfer();
DROP INDEX IF EXISTS idx_transfer_status_active;
DROP INDEX IF EXISTS idx_transfer_outflow_active;
DROP INDEX IF EXISTS idx_transfer_inflow_active;
DROP INDEX IF EXISTS uq_transfer_pair_active;
DROP INDEX IF EXISTS uq_transfer_outflow_active;
DROP INDEX IF EXISTS uq_transfer_inflow_active;

DROP TABLE IF EXISTS transfers;
DROP TYPE IF EXISTS transfer_status_enum;
-- +goose StatementEnd
