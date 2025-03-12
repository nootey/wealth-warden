-- +goose Up
-- +goose StatementBegin
CREATE TABLE monthly_budget (
id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
organization_id BIGINT UNSIGNED NOT NULL,
user_id BIGINT UNSIGNED NOT NULL,
dynamic_category_id BIGINT UNSIGNED NOT NULL, -- References a specific dynamic category
month TINYINT UNSIGNED NOT NULL CHECK (month BETWEEN 1 AND 12),
year YEAR NOT NULL,
total_inflow DECIMAL(15,2) NOT NULL, -- Sum of inflows for the period
total_outflow DECIMAL(15,2) NOT NULL, -- Sum of outflows for the period
effective_budget DECIMAL(15,2) GENERATED ALWAYS AS (total_inflow - total_outflow) STORED, -- Net result
budget_snapshot DECIMAL(15,2), -- Snapshot of effective_budget
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (dynamic_category_id) REFERENCES dynamic_categories(id),
FOREIGN KEY (organization_id) REFERENCES organizations(id),
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_org_id (organization_id),
UNIQUE (organization_id, dynamic_category_id, year, month)

);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS monthly_budget;
-- +goose StatementEnd
