-- +goose Up
-- +goose StatementBegin
CREATE TYPE export_status_enum AS ENUM ('pending', 'success', 'failed');

CREATE TABLE exports (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id BIGINT NOT NULL,
    export_type VARCHAR(128) NOT NULL,
    status export_status_enum NOT NULL DEFAULT 'pending',
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    file_path TEXT,
    file_size BIGINT,
    error TEXT,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_exports_user
     FOREIGN KEY (user_id)
         REFERENCES users(id)
         ON DELETE CASCADE
);

CREATE INDEX idx_exports_user_id ON exports(user_id);
CREATE INDEX idx_exports_status ON exports(status);

CREATE TRIGGER set_exports_updated_at
    BEFORE UPDATE ON exports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS exports;
DROP TYPE IF EXISTS export_status_enum;
-- +goose StatementEnd