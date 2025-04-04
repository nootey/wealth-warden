-- +goose Up
-- +goose StatementBegin
CREATE TABLE monthly_budget_allocations (
id BIGINT PRIMARY KEY AUTO_INCREMENT,
monthly_budget_id BIGINT UNSIGNED NOT NULL, -- Links to monthly_budget
category ENUM('savings', 'investments', 'other') NOT NULL,
allocated_value DECIMAL(15,2) NOT NULL CHECK (allocated_value >= 0),
used_value DECIMAL(15,2) DEFAULT 0,
value_method ENUM('absolute', 'percentage') NOT NULL DEFAULT 'percentage',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (monthly_budget_id) REFERENCES monthly_budget(id) ON DELETE CASCADE,
UNIQUE (monthly_budget_id, category) -- Ensures only one allocation per type per effective inflow
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS monthly_budget_allocations;
-- +goose StatementEnd
