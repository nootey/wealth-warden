-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS inflow_categories (
id INT AUTO_INCREMENT PRIMARY KEY,
user_id INT NOT NULL,
name VARCHAR(100) NOT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_user_id (user_id)  -- Index for performance on user queries
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS inflow_categories;
-- +goose StatementEnd
