-- +goose Up
-- +goose StatementBegin
CREATE TYPE import_status_enum AS ENUM ('pending', 'success', 'failed');

CREATE TABLE imports (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_id     BIGINT NOT NULL,
    account_id  BIGINT NOT NULL,
    type VARCHAR(128) NOT NULL,
    sub_type VARCHAR(128) NOT NULL,
    investments_transferred BOOLEAN,
    status import_status_enum NOT NULL DEFAULT 'pending',
    currency CHAR(3) NOT NULL DEFAULT 'EUR',
    error TEXT,
    step VARCHAR(64),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_ttpl_user     FOREIGN KEY (user_id)     REFERENCES users(id),
    CONSTRAINT fk_ttpl_account  FOREIGN KEY (account_id)  REFERENCES accounts(id)
);


CREATE TRIGGER set_imports_updated_at
    BEFORE UPDATE ON imports
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS imports;
DROP TYPE IF EXISTS import_status_enum;
-- +goose StatementEnd