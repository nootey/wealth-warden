-- +goose Up
-- +goose StatementBegin
CREATE TYPE report_status_enum AS ENUM ('pending', 'processing', 'completed', 'failed');

CREATE TABLE reports (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(128) NOT NULL,
    status report_status_enum NOT NULL DEFAULT 'pending',
    metadata JSONB,
    error TEXT,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_reports_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

CREATE INDEX idx_reports_user_id ON reports(user_id);
CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_type ON reports(type);

CREATE TRIGGER set_reports_updated_at
    BEFORE UPDATE ON reports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_reports_updated_at ON reports;
DROP TABLE IF EXISTS reports;
DROP TYPE IF EXISTS report_status_enum;
-- +goose StatementEnd
