-- +goose Up
-- +goose StatementBegin
-- Cursor pages resume from (sort value, id), so the busiest default list orders get an
-- index that ends in id; the planner can then seek to the cursor instead of scanning to it.
CREATE INDEX idx_transactions_user_txn_date_id ON transactions (user_id, txn_date, id);
CREATE INDEX idx_transfers_user_created_at_id ON transfers (user_id, created_at, id);
CREATE INDEX idx_notifications_user_created_at_id ON notifications (user_id, created_at, id);

DROP INDEX IF EXISTS idal_created_at;
CREATE INDEX idal_created_at_id ON activity_logs (created_at, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idal_created_at_id;
CREATE INDEX idal_created_at ON activity_logs (created_at);

DROP INDEX IF EXISTS idx_notifications_user_created_at_id;
DROP INDEX IF EXISTS idx_transfers_user_created_at_id;
DROP INDEX IF EXISTS idx_transactions_user_txn_date_id;
-- +goose StatementEnd
