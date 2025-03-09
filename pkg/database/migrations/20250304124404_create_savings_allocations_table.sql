-- Every time savings are allocated, a record is inserted. This lets you adjust monthly savings while keeping a history.
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS savings_allocations (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
organization_id BIGINT UNSIGNED NOT NULL,
user_id BIGINT UNSIGNED NOT NULL,
savings_category_id BIGINT UNSIGNED NOT NULL,
year DATE NOT NULL, -- Track year
month DATE NOT NULL, -- Track month
allocated_amount DECIMAL(10,2) NOT NULL, -- How much was saved that month
adjusted_amount DECIMAL(10,2) DEFAULT NULL, -- Manual user adjustment (if they modify, defaults to match allocated_amount)
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
FOREIGN KEY (savings_category_id) REFERENCES savings_categories(id) ON DELETE CASCADE,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_org_id (organization_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS savings_allocations;
-- +goose StatementEnd
