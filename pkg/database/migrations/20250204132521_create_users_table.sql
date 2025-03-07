-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
username VARCHAR(255) UNIQUE NOT NULL,
password VARCHAR(255) NOT NULL,
email VARCHAR(255) UNIQUE NOT NULL,
display_name VARCHAR(255) NOT NULL,
email_verified DATETIME,
role_id BIGINT UNSIGNED NOT NULL,
primary_organization_id BIGINT UNSIGNED NOT NULL,
FOREIGN KEY (primary_organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
deleted_at TIMESTAMP DEFAULT NULL
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
