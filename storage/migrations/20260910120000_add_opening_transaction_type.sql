-- +goose Up
-- +goose StatementBegin
-- Alone in its own migration: Postgres refuses to use a new enum value in the
-- transaction that adds it, and goose wraps every migration in one.
ALTER TYPE transaction_type_enum ADD VALUE 'opening';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- A value cannot be dropped from an enum, so the type is rebuilt without it.
-- The partial index has the old type baked into its predicate, so it goes first.
DROP INDEX IF EXISTS idx_transactions_user_date_ledger;

ALTER TYPE transaction_type_enum RENAME TO transaction_type_enum_old;

CREATE TYPE transaction_type_enum AS ENUM
  ('ledger','transfer','adjustment','trade','investment_income');

ALTER TABLE transactions ALTER COLUMN transaction_type DROP DEFAULT;
ALTER TABLE transactions ALTER COLUMN transaction_type TYPE transaction_type_enum
  USING transaction_type::text::transaction_type_enum;
ALTER TABLE transactions ALTER COLUMN transaction_type SET DEFAULT 'ledger';

DROP TYPE transaction_type_enum_old;

CREATE INDEX idx_transactions_user_date_ledger ON transactions(user_id, txn_date)
  WHERE deleted_at IS NULL AND transaction_type = 'ledger';
-- +goose StatementEnd
