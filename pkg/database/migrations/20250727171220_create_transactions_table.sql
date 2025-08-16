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

    CONSTRAINT fk_transactions_user     FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_transactions_account  FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    CONSTRAINT fk_transactions_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX idx_transactions_user_date    ON transactions(user_id, txn_date);
CREATE INDEX idx_transactions_account_date ON transactions(account_id, txn_date);
CREATE INDEX idx_transactions_category     ON transactions(category_id);

CREATE TRIGGER set_transactions_updated_at
    BEFORE UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_transactions_updated_at ON transactions;
DROP TABLE IF EXISTS transactions;
DROP TYPE IF EXISTS transaction_type_enum;
-- +goose StatementEnd
