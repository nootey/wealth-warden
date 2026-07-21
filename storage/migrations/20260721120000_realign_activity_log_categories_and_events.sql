-- +goose Up
-- +goose StatementBegin

-- Asset logs were written as `investment` but the asset trail requests `investment_asset`
UPDATE activity_logs
SET category = 'investment_asset'
WHERE category = 'investment';

-- Account activate/deactivate was written as `update`; `is_active` in the payload marks those rows
-- (UpdateAccount never diffs is_active, so it cannot collide)
UPDATE activity_logs
SET event = CASE metadata->'new'->>'is_active'
                WHEN 'true' THEN 'restore'
                ELSE 'deactivate'
            END
WHERE category = 'account'
  AND event = 'update'
  AND metadata IS NOT NULL
  AND metadata->'new'->>'is_active' IS NOT NULL;

-- User deletes stamped no subject id, so they never matched a trail; recover it via the email
UPDATE activity_logs al
SET metadata = jsonb_set(al.metadata, '{new,id}', to_jsonb(u.id::text))
FROM users u
WHERE al.category = 'user'
  AND al.event = 'delete'
  AND al.metadata IS NOT NULL
  AND al.metadata->'new'->>'id' IS NULL
  AND al.metadata->'old'->>'email' = u.email;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

UPDATE activity_logs
SET category = 'investment'
WHERE category = 'investment_asset';

UPDATE activity_logs
SET event = 'update'
WHERE category = 'account'
  AND event IN ('deactivate', 'restore');

UPDATE activity_logs
SET metadata = metadata #- '{new,id}'
WHERE category = 'user'
  AND event = 'delete'
  AND metadata IS NOT NULL;

-- +goose StatementEnd
