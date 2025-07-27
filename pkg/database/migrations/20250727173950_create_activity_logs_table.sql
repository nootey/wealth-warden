-- +goose Up
-- +goose StatementBegin
CREATE TABLE activity_logs (
id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
event       VARCHAR(255) NOT NULL COMMENT 'Log event.',
category    VARCHAR(255) NOT NULL COMMENT 'Log category.',
description TEXT NULL COMMENT 'Log description.',
metadata     JSON NULL COMMENT 'Payload of the log.',
causer_id  BIGINT UNSIGNED NULL COMMENT 'Causer logged.',
created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Date of creation.',
updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Date of update.',

INDEX idx_event (event),
INDEX idx_category (category),
INDEX idx_subject_id (causer_id),

FOREIGN KEY (causer_id) REFERENCES users(id) ON DELETE SET NULL
);
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS activity_logs;
-- +goose StatementEnd