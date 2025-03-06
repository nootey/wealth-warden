-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS permissions(
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
name varchar(75) UNIQUE NOT NULL,
description varchar(255),
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS permissions;
-- +goose StatementEnd
