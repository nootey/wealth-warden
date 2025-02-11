-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS recurring_inflows  (
id INT AUTO_INCREMENT PRIMARY KEY,
inflow_type_id INT,
amount DECIMAL(10, 2) NOT NULL,
start_date DATE NOT NULL,
end_date DATE DEFAULT NULL,  -- NULL if it continues indefinitely
frequency ENUM('daily', 'weekly', 'monthly', 'yearly') NOT NULL,
FOREIGN KEY (inflow_type_id) REFERENCES inflow_types(id),
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
deleted_at TIMESTAMP DEFAULT null
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recurring_inflows ;
-- +goose StatementEnd