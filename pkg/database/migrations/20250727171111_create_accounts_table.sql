-- +goose Up
-- +goose StatementBegin

CREATE TABLE accounts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    account_type_id BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    include_in_net_worth BOOLEAN NOT NULL DEFAULT TRUE,
    opened_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_accounts_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_accounts_account_type FOREIGN KEY (account_type_id) REFERENCES account_types(id)
);

CREATE INDEX idx_accounts_user ON accounts(user_id);

-- Allow same (user_id, name) to be reused once a prior account is closed
CREATE UNIQUE INDEX uq_accounts_name_partial
    ON accounts(user_id, name)
    WHERE closed_at IS NULL AND is_active IS TRUE;

CREATE INDEX idx_accounts_user_open
    ON accounts(user_id)
    WHERE closed_at IS NULL AND is_active IS TRUE;

CREATE INDEX idx_accounts_closed_at
    ON accounts(closed_at)
    WHERE closed_at IS NOT NULL;

-- Cannot be both enabled and closed
CREATE OR REPLACE FUNCTION prevent_conflicting_account_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.is_active = TRUE AND NEW.closed_at IS NOT NULL THEN
        RAISE EXCEPTION 'Account cannot be both active and closed at the same time';
END IF;
RETURN NEW;
END;
$$;

CREATE TRIGGER trg_accounts_state_guard
    BEFORE INSERT OR UPDATE OF is_active, closed_at ON accounts
    FOR EACH ROW
EXECUTE FUNCTION prevent_conflicting_account_state();

-- soft delete
CREATE OR REPLACE FUNCTION soft_delete_account()
RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
UPDATE accounts
SET closed_at = COALESCE(closed_at, CURRENT_TIMESTAMP),
    updated_at = CURRENT_TIMESTAMP
WHERE id = OLD.id;
RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_soft_delete_accounts ON accounts;
CREATE TRIGGER trg_soft_delete_accounts
    BEFORE DELETE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION soft_delete_account();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP RULE IF EXISTS soft_delete_accounts ON accounts;
DROP TRIGGER IF EXISTS trg_accounts_state_guard ON accounts;
DROP FUNCTION IF EXISTS prevent_conflicting_account_state();
DROP TRIGGER IF EXISTS set_accounts_updated_at ON accounts;
DROP INDEX IF EXISTS idx_accounts_closed_at;
DROP INDEX IF EXISTS idx_accounts_user_open;
DROP INDEX IF EXISTS uq_accounts_name_partial;
DROP TABLE IF EXISTS accounts;
-- +goose StatementEnd
