-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS outflows (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
outflow_category_id BIGINT UNSIGNED NOT NULL,
amount DECIMAL(10, 2) NOT NULL,
description VARCHAR(255),
outflow_date TIMESTAMP NOT NULL,
deleted_at TIMESTAMP DEFAULT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (outflow_category_id) REFERENCES outflow_categories(id) ON DELETE CASCADE,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_user_id (user_id),  -- Index for performance on user queries
INDEX idx_outflow_date (outflow_date)  -- Index for performance on date queries
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS outflows;
-- +goose StatementEnd
