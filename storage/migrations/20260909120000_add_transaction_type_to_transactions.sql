-- +goose Up
-- +goose StatementBegin
CREATE TYPE transaction_type_enum AS ENUM
  ('ledger','transfer','adjustment','trade','investment_income');

ALTER TABLE transactions
  ADD COLUMN transaction_type transaction_type_enum NOT NULL DEFAULT 'ledger';

-- The backfill has to reach soft-deleted rows too, or restoring one later hands back a
-- transaction that wrongly claims to be 'ledger'. Goose wraps this in a transaction, so
-- the trigger comes back even if a later statement fails.
ALTER TABLE transactions DISABLE TRIGGER trg_txn_block_update_if_deleted;

-- Merging two accounts flags the transfer legs between them as adjustments, so those
-- rows carry both is_transfer and is_adjustment. Adjustment is tested first because it
-- is the later, more specific state.
UPDATE transactions t SET transaction_type = CASE
  WHEN t.is_adjustment THEN 'adjustment'
  WHEN t.is_transfer   THEN 'transfer'
  WHEN EXISTS (SELECT 1 FROM investment_trades it WHERE it.transaction_id = t.id) THEN 'trade'
  WHEN EXISTS (SELECT 1 FROM investment_income ii WHERE ii.linked_transaction_id = t.id) THEN 'investment_income'
  -- Every is_system row older than the trade backfill is a dividend.
  WHEN t.is_system     THEN 'investment_income'
  ELSE 'ledger'
END::transaction_type_enum;

ALTER TABLE transactions ENABLE TRIGGER trg_txn_block_update_if_deleted;

ALTER TABLE transactions
  DROP COLUMN is_system,
  DROP COLUMN is_transfer,
  DROP COLUMN is_adjustment;

CREATE INDEX idx_transactions_user_date_ledger ON transactions(user_id, txn_date)
  WHERE deleted_at IS NULL AND transaction_type = 'ledger';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transactions_user_date_ledger;

ALTER TABLE transactions
  ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN is_transfer BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN is_adjustment BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE transactions DISABLE TRIGGER trg_txn_block_update_if_deleted;

UPDATE transactions SET
  is_transfer   = (transaction_type = 'transfer'),
  is_adjustment = (transaction_type = 'adjustment'),
  is_system     = (transaction_type IN ('trade','investment_income'));

ALTER TABLE transactions ENABLE TRIGGER trg_txn_block_update_if_deleted;

ALTER TABLE transactions DROP COLUMN transaction_type;

DROP TYPE transaction_type_enum;
-- +goose StatementEnd
