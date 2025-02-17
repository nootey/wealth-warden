-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
username VARCHAR(255) UNIQUE NOT NULL,
password VARCHAR(255) NOT NULL,
email VARCHAR(255) UNIQUE NOT NULL,
display_name VARCHAR(255),
role VARCHAR(50) DEFAULT 'member' NOT NULL,
email_verified DATETIME,
allow_log BOOLEAN DEFAULT TRUE,
last_login DATETIME,
last_login_ip VARCHAR(255),
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
deleted_at TIMESTAMP DEFAULT null
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
