-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_organizations (
user_id INT NOT NULL,
organization_id INT NOT NULL,
role VARCHAR(50) DEFAULT 'Member',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
PRIMARY KEY (user_id, organization_id),
CONSTRAINT fk_user_org_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
CONSTRAINT fk_user_org_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_organizations;
-- +goose StatementEnd
