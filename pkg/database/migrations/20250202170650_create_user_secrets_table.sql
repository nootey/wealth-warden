-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS user_secrets (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
budget_initialized BOOLEAN NOT NULL DEFAULT FALSE,
allow_log BOOLEAN DEFAULT TRUE,
last_login DATETIME,
last_login_ip VARCHAR(255),
backup_email VARCHAR(100),
two_factor_secret TEXT,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
deleted_at TIMESTAMP DEFAULT NULL,
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
INDEX idx_user_id (user_id),
INDEX idx_last_login_ip (last_login_ip)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_secrets;
-- +goose StatementEnd
