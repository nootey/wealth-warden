-- +goose Up
-- +goose StatementBegin
CREATE TYPE transaction_type_enum AS ENUM ('income', 'expense');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE transactions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    category_id BIGINT NULL,
    transaction_type transaction_type_enum NOT NULL DEFAULT 'expense',
    amount NUMERIC(19,4) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    txn_date TIMESTAMPTZ NOT NULL,
    description VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_transactions_user     FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_transactions_account  FOREIGN KEY (account_id) REFERENCES accounts(id),
    CONSTRAINT fk_transactions_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX idx_transactions_user_date_active
    ON transactions(user_id, txn_date)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_transactions_account_date_active
    ON transactions(account_id, txn_date)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_transactions_category_active
    ON transactions(category_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_transactions_deleted_at
    ON transactions(deleted_at)
    WHERE deleted_at IS NOT NULL;

-- Prevent posting into closed/deleted accounts
CREATE OR REPLACE FUNCTION prevent_txn_on_closed_or_deleted_account()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    acc_closed_at TIMESTAMPTZ;
acc_deleted_at TIMESTAMPTZ;
BEGIN
SELECT closed_at, deleted_at
INTO acc_closed_at, acc_deleted_at
FROM accounts
WHERE id = NEW.account_id;

IF acc_closed_at IS NOT NULL THEN
        RAISE EXCEPTION 'Account % is closed; cannot post transactions', NEW.account_id
            USING ERRCODE = 'check_violation';
END IF;

IF acc_deleted_at IS NOT NULL THEN
        RAISE EXCEPTION 'Account % is deleted; cannot post transactions', NEW.account_id
            USING ERRCODE = 'check_violation';
END IF;

RETURN NEW;
END;
$$;

CREATE TRIGGER trg_txn_prevent_post_to_closed
    BEFORE INSERT OR UPDATE OF account_id, amount, txn_date, transaction_type, currency
    ON transactions
    FOR EACH ROW
EXECUTE FUNCTION prevent_txn_on_closed_or_deleted_account();

-- Block any updates to already soft-deleted transactions
CREATE OR REPLACE FUNCTION prevent_update_of_soft_deleted_txn()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    old_stripped jsonb;
    new_stripped jsonb;
BEGIN
    IF OLD.deleted_at IS NOT NULL THEN
    -- Build row snapshots without deleted_at / updated_at for equality check
    old_stripped := to_jsonb(OLD) - 'deleted_at' - 'updated_at';
    new_stripped := to_jsonb(NEW) - 'deleted_at' - 'updated_at';

        -- Allow restore: only change is deleted_at -> NULL (updated_at may change)
    IF NEW.deleted_at IS NULL AND new_stripped = old_stripped THEN
          RETURN NEW;
    END IF;

    RAISE EXCEPTION 'Transaction % is soft-deleted; updates are not allowed', OLD.id
          USING ERRCODE = '25006';
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_txn_block_update_if_deleted
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION prevent_update_of_soft_deleted_txn();

-- soft delete
CREATE OR REPLACE FUNCTION soft_delete_transaction()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
UPDATE transactions
SET deleted_at = COALESCE(deleted_at, CURRENT_TIMESTAMP),
    updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_soft_delete_transactions ON transactions;
CREATE TRIGGER trg_soft_delete_transactions
    BEFORE DELETE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION soft_delete_transaction();

CREATE TRIGGER set_transactions_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Roll up transactions into balances

-- Helper to apply deltas into balances row
CREATE OR REPLACE FUNCTION balances_apply_delta(
    p_account_id BIGINT,
    p_as_of DATE,
    p_currency CHAR(3),
    p_cash_inflows NUMERIC(19,4),
    p_cash_outflows NUMERIC(19,4)
) RETURNS VOID
LANGUAGE plpgsql AS $$
BEGIN
INSERT INTO balances (account_id, as_of, cash_inflows, cash_outflows, currency)
VALUES (
           p_account_id,
           p_as_of,
           COALESCE(p_cash_inflows, 0),
           COALESCE(p_cash_outflows, 0),
           p_currency
       )
ON CONFLICT (account_id, as_of)
DO UPDATE SET
    cash_inflows  = balances.cash_inflows  + COALESCE(EXCLUDED.cash_inflows, 0),
    cash_outflows = balances.cash_outflows + COALESCE(EXCLUDED.cash_outflows, 0),
    currency      = EXCLUDED.currency,
    updated_at    = CURRENT_TIMESTAMP;
END;
$$;

-- Map a txn to (+inflow, +outflow)
CREATE OR REPLACE FUNCTION txn_effect(
    p_type transaction_type_enum,
    p_amount NUMERIC(19,4)
) RETURNS TABLE (inflow NUMERIC(19,4), outflow NUMERIC(19,4))
LANGUAGE sql IMMUTABLE AS $$
SELECT
    CASE WHEN p_type = 'income'  THEN p_amount ELSE 0 END AS inflow,
    CASE WHEN p_type = 'expense' THEN p_amount ELSE 0 END AS outflow
    $$;

-- Add effect (skip soft-deleted)
CREATE OR REPLACE FUNCTION rollup_txn_after_insert()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
DECLARE
  as_of DATE;
eff RECORD;
BEGIN
    IF NEW.deleted_at IS NOT NULL THEN
    RETURN NULL;
END IF;

as_of := (NEW.txn_date AT TIME ZONE 'UTC')::date;
SELECT * INTO eff FROM txn_effect(NEW.transaction_type, NEW.amount);

PERFORM balances_apply_delta(NEW.account_id, as_of, NEW.currency, eff.inflow, eff.outflow);
RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_transactions_rollup_insert ON transactions;
CREATE TRIGGER trg_transactions_rollup_insert
    AFTER INSERT ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION rollup_txn_after_insert();

-- Handle edits, soft-delete, restore, and moves
CREATE OR REPLACE FUNCTION rollup_txn_after_update()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
DECLARE
  old_as_of DATE := (OLD.txn_date AT TIME ZONE 'UTC')::date;
new_as_of DATE := (NEW.txn_date AT TIME ZONE 'UTC')::date;
old_eff RECORD;
new_eff RECORD;
BEGIN

-- Soft-delete: subtract old effect
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
SELECT * INTO old_eff FROM txn_effect(OLD.transaction_type, OLD.amount);
PERFORM balances_apply_delta(OLD.account_id, old_as_of, OLD.currency, -old_eff.inflow, -old_eff.outflow);
RETURN NULL;
END IF;

-- Restore: add NEW effect
IF OLD.deleted_at IS NOT NULL AND NEW.deleted_at IS NULL THEN
SELECT * INTO new_eff FROM txn_effect(NEW.transaction_type, NEW.amount);
PERFORM balances_apply_delta(NEW.account_id, new_as_of, NEW.currency, new_eff.inflow, new_eff.outflow);
RETURN NULL;
END IF;

-- Normal edit/move when both sides not deleted
IF NEW.deleted_at IS NULL AND OLD.deleted_at IS NULL THEN
    IF OLD.account_id = NEW.account_id
       AND old_as_of    = new_as_of
       AND OLD.currency = NEW.currency
       AND OLD.amount   = NEW.amount
       AND OLD.transaction_type = NEW.transaction_type THEN
       RETURN NULL;
END IF;

SELECT * INTO old_eff FROM txn_effect(OLD.transaction_type, OLD.amount);
SELECT * INTO new_eff FROM txn_effect(NEW.transaction_type, NEW.amount);

-- Reverse OLD
PERFORM balances_apply_delta(OLD.account_id, old_as_of, OLD.currency, -old_eff.inflow, -old_eff.outflow);
-- Apply NEW
PERFORM balances_apply_delta(NEW.account_id, new_as_of, NEW.currency,  new_eff.inflow,  new_eff.outflow);
RETURN NULL;
END IF;

RETURN NULL; -- both deleted: ignore
END;
$$;

DROP TRIGGER IF EXISTS trg_transactions_rollup_update ON transactions;
CREATE TRIGGER trg_transactions_rollup_update
    AFTER UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION rollup_txn_after_update();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP RULE IF EXISTS soft_delete_transactions ON transactions;
DROP TRIGGER IF EXISTS trg_txn_block_update_if_deleted ON transactions;
DROP FUNCTION IF EXISTS prevent_update_of_soft_deleted_txn();
DROP TRIGGER IF EXISTS trg_txn_prevent_post_to_closed ON transactions;
DROP FUNCTION IF EXISTS prevent_txn_on_closed_or_deleted_account();
DROP TRIGGER IF EXISTS trg_transactions_rollup_update ON transactions;
DROP TRIGGER IF EXISTS trg_transactions_rollup_insert ON transactions;
DROP FUNCTION IF EXISTS rollup_txn_after_update();
DROP FUNCTION IF EXISTS rollup_txn_after_insert();
DROP FUNCTION IF EXISTS txn_effect(transaction_type_enum, NUMERIC);
DROP FUNCTION IF EXISTS balances_apply_delta(BIGINT, DATE, CHAR(3), NUMERIC, NUMERIC);
DROP TRIGGER IF EXISTS set_transactions_updated_at ON transactions;
DROP TABLE IF EXISTS transactions;
DROP TYPE IF EXISTS transaction_type_enum;
-- +goose StatementEnd
