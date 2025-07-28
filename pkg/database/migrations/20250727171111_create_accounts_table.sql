-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
name VARCHAR(150) NOT NULL,
subtype VARCHAR(100),
classification VARCHAR(20) GENERATED ALWAYS AS (
    CASE
        WHEN subtype IN ('loan', 'credit_card', 'liability') THEN 'liability'
        ELSE 'asset'
        END
    ) STORED,
currency CHAR(3)       NOT NULL DEFAULT 'EUR',
created_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP   DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
UNIQUE (user_id, name),
INDEX idx_accounts_user (user_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS accounts;
-- +goose StatementEnd
