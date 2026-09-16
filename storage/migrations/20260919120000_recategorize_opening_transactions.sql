-- +goose Up
-- +goose StatementBegin
-- Opening transactions used to be filed under 'uncategorized'. They now default to
-- 'adjustment', matching the manual-adjustment transaction type they're partially
-- editable like. Every user already has an 'adjustment' root category (seeded as a
-- default category), but insert one for anyone who somehow lacks it before repointing.
INSERT INTO categories (user_id, name, display_name, classification, parent_id, is_default, created_at, updated_at)
SELECT DISTINCT t.user_id, 'adjustment', '(Adjustment)', 'adjustment'::category_classification, NULL::bigint, true, NOW(), NOW()
FROM transactions t
JOIN categories old_c ON old_c.id = t.category_id
WHERE t.transaction_type = 'opening'
  AND old_c.classification = 'uncategorized'
  AND NOT EXISTS (
      SELECT 1 FROM categories c
      WHERE c.user_id = t.user_id AND c.classification = 'adjustment' AND c.parent_id IS NULL
  );

UPDATE transactions t
SET category_id = c.id,
    updated_at = NOW()
FROM categories old_c, categories c
WHERE t.category_id = old_c.id
  AND old_c.classification = 'uncategorized'
  AND t.transaction_type = 'opening'
  AND c.user_id = t.user_id AND c.classification = 'adjustment' AND c.parent_id IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Every opening transaction was uncategorized before this migration - see Up - so
-- reverting is safe to apply to the whole set, not just the rows Up touched.
UPDATE transactions t
SET category_id = c.id,
    updated_at = NOW()
FROM categories c
WHERE t.transaction_type = 'opening'
  AND c.user_id = t.user_id AND c.classification = 'uncategorized' AND c.parent_id IS NULL;
-- +goose StatementEnd
