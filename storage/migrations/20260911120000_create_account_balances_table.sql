-- +goose Up
-- +goose StatementBegin
-- One row per account holding the current cash balance. Every write adds a delta;
-- transactions stay the audit trail that explains the number.
CREATE TABLE account_balances (
    account_id BIGINT        NOT NULL,
    user_id    BIGINT        NOT NULL,
    currency   CHAR(3)       NOT NULL DEFAULT 'EUR',
    balance    NUMERIC(19,4) NOT NULL DEFAULT 0,

    updated_at TIMESTAMPTZ   DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (account_id),
    CONSTRAINT fk_account_balances_account FOREIGN KEY (account_id)
        REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_account_balances_user ON account_balances(user_id);

CREATE TRIGGER set_account_balances_updated_at
    BEFORE UPDATE ON account_balances
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- The opening balance became a transaction in an earlier migration, so the sum of
-- live transactions is the whole balance. An account with none gets a zero row.
INSERT INTO account_balances (account_id, user_id, currency, balance)
SELECT a.id,
       a.user_id,
       a.currency,
       COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0)
FROM   accounts a
LEFT JOIN transactions t
       ON t.account_id = a.id AND t.deleted_at IS NULL
GROUP BY a.id, a.user_id, a.currency;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE account_balances;
-- +goose StatementEnd
