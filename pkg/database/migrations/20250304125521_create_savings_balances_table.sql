-- This stores the cumulative total for each savings category.
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS savings_balances (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
savings_category_id BIGINT UNSIGNED NOT NULL,
year INT NOT NULL, -- Yearly balance tracking
total_saved DECIMAL(10,2) NOT NULL DEFAULT 0.00, -- Total savings added this year
total_used DECIMAL(10,2) NOT NULL DEFAULT 0.00, -- Withdrawals this year
interest_earned DECIMAL(10,2) NOT NULL DEFAULT 0.00, -- Interest gained over time
reassigned_amount DECIMAL(10,2) NOT NULL DEFAULT 0.00, -- Amount reassigned to another category
balance DECIMAL(10,2) NOT NULL DEFAULT 0.00, -- Running balance (total_saved + interest - total_used - reassigned_amount)
last_updated DATE DEFAULT NULL, -- Last modification date
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (savings_category_id) REFERENCES savings_categories(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS savings_balances;
-- +goose StatementEnd
