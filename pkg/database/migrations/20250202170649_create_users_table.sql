-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'member' NOT NULL,
    email_verified DATETIME,
    allow_log BOOLEAN DEFAULT TRUE,
    last_login DATETIME,
    last_login_ip VARCHAR(255),
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME
    );

-- +goose Down
DROP TABLE IF EXISTS users;
