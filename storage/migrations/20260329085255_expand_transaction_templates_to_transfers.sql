-- +goose Up
-- +goose StatementBegin
CREATE TYPE template_type_enum AS ENUM ('transaction', 'transfer');

ALTER TABLE transaction_templates
    ADD COLUMN template_type  template_type_enum NOT NULL DEFAULT 'transaction',
    ADD COLUMN to_account_id  BIGINT NULL,
    ALTER COLUMN transaction_type DROP NOT NULL,
    ADD CONSTRAINT fk_ttpl_to_account FOREIGN KEY (to_account_id) REFERENCES accounts(id),
    ADD CONSTRAINT chk_ttpl_transfer_fields CHECK (
        (template_type = 'transaction' AND to_account_id IS NULL AND transaction_type IS NOT NULL)
        OR
        (template_type = 'transfer' AND to_account_id IS NOT NULL AND transaction_type IS NULL)
    );

CREATE INDEX idx_ttpl_to_account_active
    ON transaction_templates (to_account_id)
    WHERE is_active = TRUE AND to_account_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ttpl_to_account_active;

ALTER TABLE transaction_templates
    DROP CONSTRAINT IF EXISTS chk_ttpl_transfer_fields,
    DROP CONSTRAINT IF EXISTS fk_ttpl_to_account,
    ALTER COLUMN transaction_type SET NOT NULL,
    DROP COLUMN IF EXISTS to_account_id,
    DROP COLUMN IF EXISTS template_type;

DROP TYPE IF EXISTS template_type_enum;
-- +goose StatementEnd
