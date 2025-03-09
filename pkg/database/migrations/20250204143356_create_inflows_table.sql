-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS inflows (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
organization_id BIGINT UNSIGNED NOT NULL,
user_id BIGINT UNSIGNED NOT NULL,
inflow_category_id BIGINT UNSIGNED NOT NULL,
amount DECIMAL(10, 2) NOT NULL,
description VARCHAR(255),
inflow_date TIMESTAMP NOT NULL,
deleted_at TIMESTAMP DEFAULT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (inflow_category_id) REFERENCES inflow_categories(id) ON DELETE CASCADE,
FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_org_id (organization_id),
INDEX idx_inflow_date (inflow_date)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS inflows;
-- +goose StatementEnd
