-- +goose Up
-- +goose StatementBegin
CREATE TABLE jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    type VARCHAR(64) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    run_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    trace_ctx JSONB,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT chk_jobs_status CHECK (status IN ('pending', 'processing', 'failed'))
);

-- Claim path: oldest due pending job first. Partial index keeps it lean as failed rows accumulate.
CREATE INDEX idx_jobs_claim ON jobs (run_at) WHERE status = 'pending';

CREATE TRIGGER set_jobs_updated_at
    BEFORE UPDATE ON jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_jobs_updated_at ON jobs;
DROP TABLE IF EXISTS jobs;
-- +goose StatementEnd
