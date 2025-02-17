-- +goose Up
-- +goose StatementBegin
CREATE TABLE notification_logs (
id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id     BIGINT UNSIGNED NULL COMMENT 'Recipient of the notification.',
type        VARCHAR(255) NOT NULL COMMENT 'Type of notification (EMAIL, SMS, PUSH).',
destination VARCHAR(255) NULL COMMENT 'Email, phone number, or device token.',
status      VARCHAR(50) NOT NULL COMMENT 'Notification status (SENT, FAILED, DELIVERED).',
message     TEXT NULL COMMENT 'The notification content.',
metadata    JSON NULL COMMENT 'Extra details like error response, provider info.',
created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Timestamp when log was created.',
updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Timestamp when log was updated.',

INDEX idx_user_id (user_id),
INDEX idx_type (type),
INDEX idx_status (status),

FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notification_logs;
-- +goose StatementEnd
