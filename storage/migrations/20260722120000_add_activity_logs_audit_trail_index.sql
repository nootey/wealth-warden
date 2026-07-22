-- +goose Up
-- +goose StatementBegin
CREATE INDEX idal_audit_trail ON activity_logs (causer_id, (metadata->'new'->>'id'), created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idal_audit_trail;
-- +goose StatementEnd
