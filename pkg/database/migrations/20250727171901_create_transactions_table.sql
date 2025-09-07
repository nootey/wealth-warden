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
    is_adjustment BOOLEAN NOT NULL DEFAULT FALSE,

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
    acc_deleted_at TIMESTAMPTZ;
acc_is_active  BOOLEAN;
BEGIN
SELECT deleted_at, is_active
INTO acc_deleted_at, acc_is_active
FROM accounts
WHERE id = NEW.account_id;

IF acc_deleted_at IS NOT NULL OR acc_is_active IS FALSE THEN
        RAISE EXCEPTION 'Account % is not open (inactive or deleted); cannot post transactions', NEW.account_id
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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP RULE IF EXISTS soft_delete_transactions ON transactions;
DROP TRIGGER IF EXISTS trg_txn_block_update_if_deleted ON transactions;
DROP FUNCTION IF EXISTS prevent_update_of_soft_deleted_txn();
DROP TRIGGER IF EXISTS trg_txn_prevent_post_to_closed ON transactions;
DROP FUNCTION IF EXISTS prevent_txn_on_closed_or_deleted_account();
DROP TRIGGER IF EXISTS set_transactions_updated_at ON transactions;
DROP TABLE IF EXISTS transactions;
DROP TYPE IF EXISTS transaction_type_enum;
-- +goose StatementEnd
