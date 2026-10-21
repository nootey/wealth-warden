-- +goose Up
-- +goose StatementBegin

-- Category stays optional in the client and the API DTO, but the write path always
-- resolves a concrete category (a matched rule, or the user's "(Uncategorized)" root).
-- This makes the DB match that reality: transactions.category_id becomes NOT NULL.
-- Three things must happen before the column can be locked down.

-- 1. Ensure every user with an uncategorized transaction owns an "(Uncategorized)"
--    root, mirroring EnsureRootCategory's lookup (user_id, classification =
--    'uncategorized', parent_id IS NULL). Runtime and seeding normally create it, but
--    a user could hold NULL-category rows without one yet.
INSERT INTO categories (user_id, name, display_name, classification, parent_id, is_default)
SELECT DISTINCT t.user_id, '(uncategorized)', '(Uncategorized)', 'uncategorized'::category_classification, NULL::bigint, true
FROM transactions t
WHERE t.category_id IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM categories c
      WHERE c.user_id = t.user_id
        AND c.classification = 'uncategorized'
        AND c.parent_id IS NULL
  );

-- 2. Backfill NULL category_id onto that root. trg_txn_block_update_if_deleted aborts
--    any non-restore update to a soft-deleted transaction, so a soft-deleted
--    NULL-category row would fail. Disable it for just this repoint, inside the same
--    migration transaction, and put it straight back (same pattern as 20260918100000).
ALTER TABLE transactions DISABLE TRIGGER trg_txn_block_update_if_deleted;

UPDATE transactions t
SET category_id = uc.id
FROM categories uc
WHERE t.category_id IS NULL
  AND uc.user_id = t.user_id
  AND uc.classification = 'uncategorized'
  AND uc.parent_id IS NULL;

ALTER TABLE transactions ENABLE TRIGGER trg_txn_block_update_if_deleted;

-- Abort loudly if any NULL survived, rather than let SET NOT NULL below fail obscurely.
DO $$
DECLARE
    leftover BIGINT;
BEGIN
    SELECT COUNT(*) INTO leftover FROM transactions WHERE category_id IS NULL;
    IF leftover > 0 THEN
        RAISE EXCEPTION 'category backfill incomplete: % transaction(s) still have NULL category_id', leftover;
    END IF;
END $$;

-- 3. ON DELETE SET NULL can no longer coexist with a NOT NULL column: deleting a
--    still-referenced category would try to null the column and fail. Switch to
--    RESTRICT so the DB refuses to delete a referenced category. The app already
--    blocks deleting a category with active transactions; RESTRICT also covers
--    soft-deleted rows.
ALTER TABLE transactions DROP CONSTRAINT fk_transactions_category;
ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_category
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT;

ALTER TABLE transactions ALTER COLUMN category_id SET NOT NULL;

-- 4. Align transaction_templates: drop its ON DELETE SET NULL so a category is never
--    silently nulled out from under a template. category_id stays nullable here (a
--    template need not carry a category), but a referenced category can no longer be
--    deleted out from under it.
ALTER TABLE transaction_templates DROP CONSTRAINT fk_ttpl_category;
ALTER TABLE transaction_templates
    ADD CONSTRAINT fk_ttpl_category
    FOREIGN KEY (category_id) REFERENCES categories(id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transaction_templates DROP CONSTRAINT fk_ttpl_category;
ALTER TABLE transaction_templates
    ADD CONSTRAINT fk_ttpl_category
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;

ALTER TABLE transactions ALTER COLUMN category_id DROP NOT NULL;

ALTER TABLE transactions DROP CONSTRAINT fk_transactions_category;
ALTER TABLE transactions
    ADD CONSTRAINT fk_transactions_category
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL;
-- The backfill is not reversed: a NULL replaced by the uncategorized root is not
-- recoverable, since the original NULL is gone.
-- +goose StatementEnd
