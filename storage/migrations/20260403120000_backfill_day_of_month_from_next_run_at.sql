-- +goose Up
-- +goose StatementBegin
UPDATE transaction_templates tt
SET day_of_month = EXTRACT(DAY FROM tt.next_run_at AT TIME ZONE COALESCE(su.timezone, 'UTC'))::int
FROM settings_user su
WHERE su.user_id = tt.user_id;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE transaction_templates SET day_of_month = 1;
-- +goose StatementEnd
