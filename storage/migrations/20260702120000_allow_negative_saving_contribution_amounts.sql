-- +goose Up
-- +goose StatementBegin
ALTER TABLE saving_contributions DROP CONSTRAINT saving_contributions_amount_check;
ALTER TABLE saving_contributions ADD CONSTRAINT saving_contributions_amount_check CHECK (amount <> 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE saving_contributions DROP CONSTRAINT saving_contributions_amount_check;
ALTER TABLE saving_contributions ADD CONSTRAINT saving_contributions_amount_check CHECK (amount > 0);
-- +goose StatementEnd
