-- +goose Up
-- +goose StatementBegin
-- Template runs never stored their template, only an "Auto: <name>" or "Early: <name>" description.
-- Soft-deleted rows are skipped because a trigger blocks any update to them.
UPDATE transactions t
SET template_id = tt.id
FROM transaction_templates tt
WHERE t.user_id = tt.user_id
  AND t.deleted_at IS NULL
  AND t.template_id IS NULL
  AND t.transaction_type IN ('ledger', 'transfer')
  AND t.account_id IN (tt.account_id, tt.to_account_id)
  AND t.description IN ('Auto: ' || tt.name, 'Early: ' || tt.name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE transactions SET template_id = NULL WHERE template_id IS NOT NULL AND deleted_at IS NULL;
-- +goose StatementEnd
