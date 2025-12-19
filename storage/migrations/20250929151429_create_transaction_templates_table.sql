-- +goose Up
-- +goose StatementBegin
CREATE TYPE frequency_enum AS ENUM ('weekly', 'biweekly', 'monthly', 'quarterly', 'annually');

CREATE TABLE transaction_templates (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    user_id     BIGINT NOT NULL,
    account_id  BIGINT NOT NULL,
    category_id BIGINT NULL,
    transaction_type transaction_type_enum NOT NULL DEFAULT 'expense',
    amount      NUMERIC(19,4) NOT NULL,
    frequency      frequency_enum NOT NULL,
    next_run_at TIMESTAMPTZ NOT NULL,
    last_run_at TIMESTAMPTZ NULL,
    run_count   INTEGER NOT NULL DEFAULT 0,
    end_date    DATE NULL,
    max_runs    INTEGER NULL CHECK (max_runs > 0),

    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_ttpl_user     FOREIGN KEY (user_id)     REFERENCES users(id),
    CONSTRAINT fk_ttpl_account  FOREIGN KEY (account_id)  REFERENCES accounts(id),
    CONSTRAINT fk_ttpl_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
);

CREATE INDEX idx_ttpl_due
    ON transaction_templates (next_run_at)
    WHERE is_active = TRUE;

CREATE INDEX idx_ttpl_user_active_next
    ON transaction_templates (user_id, next_run_at)
    WHERE is_active = TRUE;

CREATE INDEX idx_ttpl_account_active
    ON transaction_templates (account_id)
    WHERE is_active = TRUE;

CREATE TRIGGER set_transaction_templates_updated_at
    BEFORE UPDATE ON transaction_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_templates;
DROP TYPE IF EXISTS frequency_enum;
-- +goose StatementEnd
