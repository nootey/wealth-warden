-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS investment_categories (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
organization_id BIGINT UNSIGNED NOT NULL,
user_id BIGINT UNSIGNED NOT NULL,
name VARCHAR(100) NOT NULL,
investment_type VARCHAR(128) NOT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_org_id (organization_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS investment_categories;
-- +goose StatementEnd