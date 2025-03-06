-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS savings_categories (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
organization_id BIGINT UNSIGNED NOT NULL,
name VARCHAR(100) NOT NULL,
savings_type ENUM('fixed', 'variable') NOT NULL,
priority INT DEFAULT 1,
goal_value DECIMAL(10,2) DEFAULT NULL, -- Target savings goal
goal_progress DECIMAL(10,2) DEFAULT 0.00, -- Tracks progress toward goal
goal_time_limit DATE DEFAULT NULL, -- Deadline for goal completion
interest_rate DECIMAL(5,2) DEFAULT NULL, -- Interest rate for this savings account
accrued_interest DECIMAL(10,2) DEFAULT 0.00, -- Earned interest (updated yearly or monthly)
account_type VARCHAR(128) DEFAULT 'manual',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
INDEX idx_org_id (organization_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS savings_categories;
-- +goose StatementEnd