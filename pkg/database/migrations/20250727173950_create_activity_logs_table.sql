-- +goose Up
-- +goose StatementBegin
CREATE TABLE activity_logs (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event       VARCHAR(255) NOT NULL,
    category    VARCHAR(255) NOT NULL,
    description TEXT NULL,
    metadata    JSONB NULL,
    causer_id   BIGINT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    FOREIGN KEY (causer_id) REFERENCES users(id)
);

-- Indexes
CREATE INDEX idal_event ON activity_logs (event, created_at);
CREATE INDEX idal_category ON activity_logs (category, created_at);
CREATE INDEX idal_causer_id ON activity_logs (causer_id, created_at);
CREATE INDEX idal_created_at ON activity_logs (created_at);

CREATE TRIGGER set_activity_logs_updated_at
    BEFORE UPDATE ON activity_logs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_activity_logs_updated_at ON activity_logs;
DROP TABLE IF EXISTS activity_logs;
-- +goose StatementEnd
