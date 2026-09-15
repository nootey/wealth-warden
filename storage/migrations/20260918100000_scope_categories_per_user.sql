-- +goose Up
-- +goose StatementBegin

-- Swap the unique constraint first, before any per-user rows exist. The old
-- (name, classification) constraint has no user_id in it at all - that's the bug this
-- whole migration exists to fix - so it would reject user 2's "Salary" as a duplicate
-- of user 1's "Salary" the moment the fan-out below starts inserting. The new
-- constraint is safe to add early: Postgres treats every NULL user_id as distinct, so
-- it does not conflict with the still-present shared default rows.
ALTER TABLE categories DROP CONSTRAINT uq_categories_name_class;
ALTER TABLE categories ADD CONSTRAINT uq_categories_user_name_class UNIQUE (user_id, name, classification);

-- Give every existing user their own copy of the root (classification) default categories.
INSERT INTO categories (user_id, name, display_name, classification, parent_id, is_default, deleted_at, created_at, updated_at)
SELECT u.id, gc.name, gc.display_name, gc.classification, NULL, true, gc.deleted_at, now(), now()
FROM users u
CROSS JOIN categories gc
WHERE gc.user_id IS NULL AND gc.is_default = true AND gc.parent_id IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM categories uc
      WHERE uc.user_id = u.id AND uc.name = gc.name AND uc.classification = gc.classification
  );

-- Give every existing user their own copy of the default subcategories, parented to the root
-- copy just created for that same user above.
INSERT INTO categories (user_id, name, display_name, classification, parent_id, is_default, deleted_at, created_at, updated_at)
SELECT u.id, gc.name, gc.display_name, gc.classification, ur.id, true, gc.deleted_at, now(), now()
FROM users u
CROSS JOIN categories gc
JOIN categories gp ON gp.id = gc.parent_id AND gp.user_id IS NULL
JOIN categories ur ON ur.user_id = u.id AND ur.name = gp.name AND ur.classification = gp.classification
WHERE gc.user_id IS NULL AND gc.is_default = true AND gc.parent_id IS NOT NULL
  AND NOT EXISTS (
      SELECT 1 FROM categories uc
      WHERE uc.user_id = u.id AND uc.name = gc.name AND uc.classification = gc.classification
  );

-- Every category a user ever created through the app - not just the defaults above -
-- was parented to the shared global root for its classification (that's how InsertCategory
-- resolved a parent before this migration's app-code counterpart). Repoint those existing
-- categories onto the user's own new root copy, or the final root deletion below fails
-- with a foreign key violation - it's still referenced.
UPDATE categories c
SET parent_id = ur.id
FROM categories gp, categories ur
WHERE c.parent_id = gp.id AND gp.user_id IS NULL AND gp.parent_id IS NULL
  AND c.user_id IS NOT NULL
  AND ur.user_id = c.user_id AND ur.parent_id IS NULL AND ur.classification = gp.classification;

-- Repoint transactions off the shared default rows and onto each user's own copy.
-- (Comma-style FROM, not a nested JOIN: the target table can't be referenced
-- inside a JOIN's ON clause, only in WHERE.)
-- trg_txn_block_update_if_deleted blocks any update to an already soft-deleted
-- transaction that isn't a pure restore, so a deleted transaction still pointing at a
-- shared default category would abort this update. Disable it for just this repoint,
-- inside the same migration transaction, and put it straight back.
ALTER TABLE transactions DISABLE TRIGGER trg_txn_block_update_if_deleted;

UPDATE transactions t
SET category_id = uc.id
FROM categories gc, categories uc
WHERE t.category_id = gc.id AND gc.user_id IS NULL AND gc.is_default = true
  AND uc.user_id = t.user_id AND uc.name = gc.name AND uc.classification = gc.classification;

ALTER TABLE transactions ENABLE TRIGGER trg_txn_block_update_if_deleted;

-- Repoint recurring transaction templates the same way.
UPDATE transaction_templates tt
SET category_id = uc.id
FROM categories gc, categories uc
WHERE tt.category_id = gc.id AND gc.user_id IS NULL AND gc.is_default = true
  AND uc.user_id = tt.user_id AND uc.name = gc.name AND uc.classification = gc.classification;

-- A group can only reference a given category once (uq_group_category). If a group already
-- has a membership row for the user's own copy, drop the row pointing at the shared default
-- before repointing, so the update below never collides with it.
DELETE FROM category_group_members m
USING categories gc, category_groups g, categories uc
WHERE m.category_id = gc.id
  AND gc.user_id IS NULL AND gc.is_default = true
  AND g.id = m.group_id
  AND uc.user_id = g.user_id AND uc.name = gc.name AND uc.classification = gc.classification
  AND EXISTS (
      SELECT 1 FROM category_group_members m2
      WHERE m2.group_id = m.group_id AND m2.category_id = uc.id
  );

-- Repoint remaining category group memberships.
UPDATE category_group_members m
SET category_id = uc.id
FROM categories gc, category_groups g, categories uc
WHERE m.category_id = gc.id
  AND gc.user_id IS NULL AND gc.is_default = true
  AND g.id = m.group_id
  AND uc.user_id = g.user_id AND uc.name = gc.name AND uc.classification = gc.classification;

-- Repoint rule "set_category" actions. value is free text, so isolate the numeric,
-- set_category rows in a subquery before casting, to avoid a bad cast on unrelated rows.
UPDATE rule_actions ra
SET value = uc.id::text
FROM (
    SELECT id, rule_id, value::bigint AS old_category_id
    FROM rule_actions
    WHERE action_type = 'set_category' AND value ~ '^\d+$'
) eligible
JOIN rules r ON r.id = eligible.rule_id
JOIN categories gc ON gc.id = eligible.old_category_id AND gc.user_id IS NULL AND gc.is_default = true
JOIN categories uc ON uc.user_id = r.user_id AND uc.name = gc.name AND uc.classification = gc.classification
WHERE ra.id = eligible.id;

-- Every known reference to a shared default category must be gone by now. If not, abort
-- rather than silently losing data to the ON DELETE SET NULL / CASCADE below.
DO $$
DECLARE
    leftover BIGINT;
BEGIN
    SELECT COUNT(*) INTO leftover FROM (
        SELECT category_id FROM transactions
            WHERE category_id IN (SELECT id FROM categories WHERE user_id IS NULL AND is_default = true)
        UNION ALL
        SELECT category_id FROM transaction_templates
            WHERE category_id IN (SELECT id FROM categories WHERE user_id IS NULL AND is_default = true)
        UNION ALL
        SELECT category_id FROM category_group_members
            WHERE category_id IN (SELECT id FROM categories WHERE user_id IS NULL AND is_default = true)
        UNION ALL
        SELECT gc.id
        FROM (
            SELECT value::bigint AS cat_id FROM rule_actions
            WHERE action_type = 'set_category' AND value ~ '^\d+$'
        ) eligible
        JOIN categories gc ON gc.id = eligible.cat_id
        WHERE gc.user_id IS NULL AND gc.is_default = true
    ) leftovers;

    IF leftover > 0 THEN
        RAISE EXCEPTION 'category repoint incomplete: % reference(s) to shared default categories remain', leftover;
    END IF;
END $$;

-- The shared default rows are now fully retired. fk_categories_parent is self-
-- referential (parent_id -> categories.id) with NO ACTION, and a single DELETE
-- covering both a root and its own children isn't guaranteed to clear the child
-- before the parent - it depends on scan order, which is why this passed on a small
-- test dataset but failed on real data. Delete children before roots explicitly.
DELETE FROM categories WHERE user_id IS NULL AND is_default = true AND parent_id IS NOT NULL;
DELETE FROM categories WHERE user_id IS NULL AND is_default = true;

ALTER TABLE categories ALTER COLUMN user_id SET NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Schema only. The per-user fan-out above is not reversible; re-run the SeedCategories
-- seeder afterwards if shared default rows are needed again.
ALTER TABLE categories DROP CONSTRAINT IF EXISTS uq_categories_user_name_class;
ALTER TABLE categories ADD CONSTRAINT uq_categories_name_class UNIQUE (name, classification);
ALTER TABLE categories ALTER COLUMN user_id DROP NOT NULL;
-- +goose StatementEnd
