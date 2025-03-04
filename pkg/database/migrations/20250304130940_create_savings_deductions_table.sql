-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS savings_deductions (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
savings_category_id BIGINT UNSIGNED NOT NULL,
deduction_date DATE NOT NULL,
amount DECIMAL(10,2) NOT NULL,
reason TEXT DEFAULT NULL, -- Optional: Reason for withdrawal
reassigned_to_category_id BIGINT UNSIGNED DEFAULT NULL, -- If reassigned to another category
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
FOREIGN KEY (savings_category_id) REFERENCES savings_categories(id) ON DELETE CASCADE,
FOREIGN KEY (reassigned_to_category_id) REFERENCES savings_categories(id) ON DELETE SET NULL -- NULL if just withdrawn
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS savings_deductions;
-- +goose StatementEnd
