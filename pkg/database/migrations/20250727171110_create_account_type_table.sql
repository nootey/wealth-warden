-- +goose Up
-- +goose StatementBegin
CREATE TABLE account_types (
    id BIGSERIAL PRIMARY KEY,
    type VARCHAR(100) NOT NULL,
    subtype VARCHAR(100) NOT NULL,
    classification VARCHAR(20) GENERATED ALWAYS AS (
              CASE
                  WHEN type IN ('loan', 'credit_card', 'other_liability') THEN 'liability'
                  ELSE 'asset'
                  END
              ) STORED,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT uq_account_type UNIQUE (type, subtype)
);

CREATE TRIGGER set_account_types_updated_at
    BEFORE UPDATE ON account_types
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_account_types_updated_at ON account_types;
DROP TABLE IF EXISTS account_types;
-- +goose StatementEnd
