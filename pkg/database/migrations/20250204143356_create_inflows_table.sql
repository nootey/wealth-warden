-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS inflows (
id INT AUTO_INCREMENT PRIMARY KEY,
inflow_type_id INT,
amount DECIMAL(10, 2) NOT NULL,
inflow_date DATE NOT NULL,
FOREIGN KEY (inflow_type_id) REFERENCES inflow_types(id),
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS inflows;
-- +goose StatementEnd