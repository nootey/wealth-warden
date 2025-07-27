-- +goose Up
-- +goose StatementBegin
CREATE TABLE categories (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
name VARCHAR(100) NOT NULL,
classification ENUM('income', 'expense','savings','investment') NOT NULL DEFAULT 'expense',
parent_id BIGINT UNSIGNED NULL,     -- for nesting
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id)    REFERENCES users(id)      ON DELETE CASCADE,
FOREIGN KEY (parent_id)  REFERENCES categories(id)  ON DELETE SET NULL,
UNIQUE (user_id, name, classification),
INDEX idx_categories_user_class (user_id, classification)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS categories;
-- +goose StatementEnd
