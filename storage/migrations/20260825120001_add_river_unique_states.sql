-- Vendored from River v0.44.1. Regenerate with:
--   river migrate-get --line main --all --exclude-version 1 --up
--   river migrate-get --line main --all --exclude-version 1 --down
-- Version 1 is excluded: it creates river_migration, and goose already tracks
-- versions. Split at the 005/006 boundary because version 004 adds the `pending`
-- enum value that 006 uses, which Postgres 16 rejects in one transaction
-- (SQLSTATE 55P04). Statements stay unwrapped so goose sends them one at a time;
-- only dollar-quoted bodies are wrapped, as their semicolons would split them.
-- This file carries River main versions 006-007.

-- +goose Up
-- +goose StatementBegin
-- River main migration 006 [up]
CREATE OR REPLACE FUNCTION river_job_state_in_bitmask(bitmask BIT(8), state river_job_state)
RETURNS boolean
LANGUAGE SQL
IMMUTABLE
AS $$
    SELECT CASE state
        WHEN 'available' THEN get_bit(bitmask, 7)
        WHEN 'cancelled' THEN get_bit(bitmask, 6)
        WHEN 'completed' THEN get_bit(bitmask, 5)
        WHEN 'discarded' THEN get_bit(bitmask, 4)
        WHEN 'pending'   THEN get_bit(bitmask, 3)
        WHEN 'retryable' THEN get_bit(bitmask, 2)
        WHEN 'running'   THEN get_bit(bitmask, 1)
        WHEN 'scheduled' THEN get_bit(bitmask, 0)
        ELSE 0
    END = 1;
$$;
-- +goose StatementEnd

--
-- Add `river_job.unique_states` and bring up an index on it.
--
-- This column may exist already if users manually created the column and index
-- as instructed in the changelog so the index could be created `CONCURRENTLY`.
--
ALTER TABLE river_job ADD COLUMN IF NOT EXISTS unique_states BIT(8);

-- This statement uses `IF NOT EXISTS` to allow users with a `river_job` table
-- of non-trivial size to build the index `CONCURRENTLY` out of band of this
-- migration, then follow by completing the migration.
CREATE UNIQUE INDEX IF NOT EXISTS river_job_unique_idx ON river_job (unique_key)
    WHERE unique_key IS NOT NULL
      AND unique_states IS NOT NULL
      AND river_job_state_in_bitmask(unique_states, state);

-- Remove the old unique index. Users who are actively using the unique jobs
-- feature and who wish to avoid deploy downtime may want od drop this in a
-- subsequent migration once all jobs using the old unique system have been
-- completed (i.e. no more rows with non-null unique_key and null
-- unique_states).
DROP INDEX river_job_kind_unique_key_idx;

-- River main migration 007 [up]
--
-- Notification outbox.
--

CREATE TABLE river_notification (
    id bigserial PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    payload text NOT NULL,
    topic text NOT NULL,
    CONSTRAINT topic_length CHECK (length(topic) > 0 AND length(topic) < 128)
);

CREATE INDEX river_notification_created_at_idx ON river_notification (created_at);

CREATE INDEX river_notification_topic_id_idx ON river_notification (topic, id);

--
-- SQLite JSONB conversion.
--
-- No-op. PostgreSQL already stores River JSON columns as jsonb.

--
-- SQL cleanup.
--

--
-- Drop unused tables `river_client` and `river_client_queue`.
--

DROP TABLE river_client_queue;

DROP TABLE river_client;

--
-- Adds `DEFAULT 25` to `river_job.max_attempts`.
--

ALTER TABLE river_job
    ALTER COLUMN max_attempts SET DEFAULT 25;

--
-- Changes `river_queue.updated_at` to have a default of `CURRENT_TIMESTAMP`.
--

ALTER TABLE river_queue
    ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;

-- +goose Down
-- River main migration 007 [down]
--
-- SQL cleanup rollback.
--

--
-- Add back unused tables `river_client` and `river_client_queue`.
--

CREATE UNLOGGED TABLE river_client (
    id text PRIMARY KEY NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    metadata jsonb NOT NULL DEFAULT '{}',
    paused_at timestamptz,
    updated_at timestamptz NOT NULL,
    CONSTRAINT name_length CHECK (char_length(id) > 0 AND char_length(id) < 128)
);

CREATE UNLOGGED TABLE river_client_queue (
    river_client_id text NOT NULL REFERENCES river_client (id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    max_workers bigint NOT NULL DEFAULT 0,
    metadata jsonb NOT NULL DEFAULT '{}',
    num_jobs_completed bigint NOT NULL DEFAULT 0,
    num_jobs_running bigint NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (river_client_id, name),
    CONSTRAINT name_length CHECK (char_length(name) > 0 AND char_length(name) < 128),
    CONSTRAINT num_jobs_completed_zero_or_positive CHECK (num_jobs_completed >= 0),
    CONSTRAINT num_jobs_running_zero_or_positive CHECK (num_jobs_running >= 0)
);

--
-- Revert addition of `DEFAULT 25` to `river_job.max_attempts`.
--

ALTER TABLE river_job
    ALTER COLUMN max_attempts DROP DEFAULT;

--
-- Changes `river_queue.updated_at` to revert the default of `CURRENT_TIMESTAMP`.
--

ALTER TABLE river_queue
    ALTER COLUMN updated_at DROP DEFAULT;

--
-- SQLite JSONB conversion rollback.
--
-- No-op. PostgreSQL already stores River JSON columns as jsonb.

--
-- Notification outbox rollback.
--

DROP TABLE river_notification;

-- River main migration 006 [down]
--
-- Drop `river_job.unique_states` and its index.
--

DROP INDEX river_job_unique_idx;

ALTER TABLE river_job
    DROP COLUMN unique_states;

CREATE UNIQUE INDEX IF NOT EXISTS river_job_kind_unique_key_idx ON river_job (kind, unique_key) WHERE unique_key IS NOT NULL;

--
-- Drop `river_job_state_in_bitmask` function.
--
DROP FUNCTION river_job_state_in_bitmask;
