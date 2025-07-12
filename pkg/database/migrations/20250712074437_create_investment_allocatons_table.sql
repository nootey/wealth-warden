-- Every time savings are allocated, a record is inserted. This lets you adjust monthly savings while keeping a history.
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS investment_allocations (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
organization_id BIGINT UNSIGNED NOT NULL,
user_id BIGINT UNSIGNED NOT NULL,
investment_category_id BIGINT UNSIGNED NOT NULL,
transaction_date TIMESTAMP NOT NULL,
allocated_amount DECIMAL(10,2) NOT NULL,
description VARCHAR(255),
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
FOREIGN KEY (investment_category_id) REFERENCES investment_categories(id) ON DELETE CASCADE,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_org_id (organization_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS investment_allocations;
-- +goose StatementEnd
