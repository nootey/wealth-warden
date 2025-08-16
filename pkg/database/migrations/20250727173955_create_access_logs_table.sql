-- +goose Up
-- +goose StatementBegin
CREATE TABLE access_logs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event       VARCHAR(255) NOT NULL,
    status      VARCHAR(255) NOT NULL,
    service     VARCHAR(255),
    ip_address  VARCHAR(50),
    user_agent  TEXT,
    causer_id   BIGINT NULL,
    description TEXT NULL,
    metadata    JSONB NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),

    FOREIGN KEY (causer_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX idac_event ON access_logs (event, created_at);
CREATE INDEX idac_status ON access_logs (status, created_at);
CREATE INDEX idac_causer_id ON access_logs (causer_id, created_at);
CREATE INDEX idac_created_at ON access_logs (created_at DESC);

CREATE TRIGGER set_access_logs_updated_at
BEFORE UPDATE ON access_logs
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_access_logs_updated_at ON access_logs;
DROP TABLE IF EXISTS access_logs;
-- +goose StatementEnd
