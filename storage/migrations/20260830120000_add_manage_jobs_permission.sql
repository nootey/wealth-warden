-- +goose Up
-- +goose StatementBegin
INSERT INTO permissions (name, description, created_at, updated_at)
VALUES ('manage_jobs', 'Manage durable queue jobs: retry, cancel, delete (global)', now(), now())
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id, created_at, updated_at)
SELECT r.id, p.id, now(), now()
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin' AND p.name = 'manage_jobs'
ON CONFLICT (role_id, permission_id) DO NOTHING;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM role_permissions
WHERE permission_id = (SELECT id FROM permissions WHERE name = 'manage_jobs');
DELETE FROM permissions WHERE name = 'manage_jobs';
-- +goose StatementEnd
