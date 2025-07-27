-- +goose Up
-- +goose StatementBegin
CREATE TABLE transactions (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
account_id BIGINT UNSIGNED NOT NULL,
category_id BIGINT UNSIGNED NULL,
transaction_type ENUM('increase','decrease','adjustment','transfer') NOT NULL DEFAULT 'decrease',
amount DECIMAL(19,4) NOT NULL,
currency CHAR(3)       NOT NULL DEFAULT 'USD',
txn_date DATE          NOT NULL,
description VARCHAR(255),
reference_id BIGINT UNSIGNED NULL,  -- for linking transfers
created_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

FOREIGN KEY (user_id)     REFERENCES users(id)     ON DELETE CASCADE,
FOREIGN KEY (account_id)  REFERENCES accounts(id)  ON DELETE CASCADE,
FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL,
FOREIGN KEY (reference_id) REFERENCES transactions(id) ON DELETE CASCADE,

INDEX idx_transactions_user_date    (user_id,    txn_date),
INDEX idx_transactions_account_date (account_id, txn_date),
INDEX idx_transactions_category     (category_id),
INDEX idx_transactions_reference    (reference_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transactions;
-- +goose StatementEnd
