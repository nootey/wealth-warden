-- +goose Up
-- +goose StatementBegin
CREATE TABLE access_logs (
id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
event       VARCHAR(255) NOT NULL COMMENT 'Event logged.',
service     VARCHAR(255) NULL COMMENT 'Service accessed (frontend, handlers ...).',
status      VARCHAR(255) NOT NULL COMMENT 'Status of attempted access.',
ip_address  VARCHAR(45) NULL COMMENT 'IP of attempted access.',
user_agent  TEXT NULL COMMENT 'User agent of attempted access.',
causer_id   BIGINT UNSIGNED NULL COMMENT 'Causer logged.',
description TEXT NULL COMMENT 'Log description.',
metadata    JSON NULL COMMENT 'Payload of the log.',
created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Date of creation.',
updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Date of update.',

INDEX idx_event (event),
INDEX idx_status (status),
INDEX idx_causer_id (causer_id),

FOREIGN KEY (causer_id) REFERENCES users(id) ON DELETE SET NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS access_logs;
-- +goose StatementEnd
