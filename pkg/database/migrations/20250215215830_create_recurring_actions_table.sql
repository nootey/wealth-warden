-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recurring_actions (
id INT AUTO_INCREMENT PRIMARY KEY,
user_id INT NOT NULL,
category_type VARCHAR(50) NOT NULL, -- Now a generic category type instead of ENUM
category_id INT NOT NULL, -- Renamed from action_id
amount DECIMAL(10, 2) NOT NULL,
start_date DATE NOT NULL,
end_date DATE DEFAULT NULL,
interval_value INT NOT NULL,
interval_unit ENUM('days', 'weeks', 'months', 'years') NOT NULL,
last_processed DATE DEFAULT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recurring_actions;
-- +goose StatementEnd
