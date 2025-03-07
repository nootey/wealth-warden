-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS role_permissions(
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
role_id BIGINT UNSIGNED NOT NULL,
permission_id BIGINT UNSIGNED NOT NULL,
FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
UNIQUE (role_id, permission_id),
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS role_permissions;
-- +goose StatementEnd
