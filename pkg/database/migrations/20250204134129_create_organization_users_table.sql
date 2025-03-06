-- +goose Up
-- +goose StatementBegin
CREATE TABLE organization_users (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
user_id BIGINT UNSIGNED NOT NULL,
organization_id BIGINT UNSIGNED NOT NULL,
role_id BIGINT UNSIGNED NOT NULL,
deleted_at TIMESTAMP DEFAULT null,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
CONSTRAINT fk_user_org_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
CONSTRAINT fk_user_org_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
CONSTRAINT fk_user_org_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
UNIQUE (user_id, organization_id)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE organization_users;
-- +goose StatementEnd
