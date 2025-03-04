-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recurring_actions (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
category_type VARCHAR(50) NOT NULL, -- Distinguish category
category_id BIGINT UNSIGNED NOT NULL, -- Map category
amount DECIMAL(10,2) DEFAULT NULL, -- Can be NULL if using percentage for savings
percentage DECIMAL(5,2) DEFAULT NULL, -- For fixed savings categories
start_date DATE NOT NULL,
end_date DATE DEFAULT NULL, -- NULL means indefinite
interval_value INT NOT NULL, -- Recurrence value (e.g., 1 for monthly)
interval_unit ENUM('days', 'weeks', 'months', 'years') NOT NULL, -- Recurrence unit
last_processed DATE DEFAULT NULL, -- Tracks last execution date
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_user_id (user_id)  -- Index for performance on user queries
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recurring_actions;
-- +goose StatementEnd
