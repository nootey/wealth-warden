-- +goose Up
-- +goose StatementBegin

CREATE TABLE accounts (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    account_type_id BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_accounts_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_accounts_account_type FOREIGN KEY (account_type_id) REFERENCES account_types(id),
    CONSTRAINT uq_accounts_name UNIQUE (user_id, name)
);

CREATE INDEX idx_accounts_user ON accounts(user_id);

CREATE TRIGGER set_accounts_updated_at
    BEFORE UPDATE ON accounts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_accounts_updated_at ON accounts;
DROP TABLE IF EXISTS accounts;
-- +goose StatementEnd
