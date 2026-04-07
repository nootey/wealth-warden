-- +goose Up
-- +goose StatementBegin
CREATE TYPE saving_goal_status AS ENUM ('active', 'paused', 'completed', 'archived');

CREATE TABLE saving_goals (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id        BIGINT NOT NULL,
    account_id     BIGINT NOT NULL,
    name           VARCHAR(150) NOT NULL,
    target_amount      NUMERIC(19,4) NOT NULL CHECK (target_amount > 0),
    current_amount     NUMERIC(19,4) NOT NULL DEFAULT 0,
    target_date        DATE NULL,
    status             saving_goal_status NOT NULL DEFAULT 'active',
    priority           INT NOT NULL DEFAULT 0,
    monthly_allocation NUMERIC(19,4) NULL CHECK (monthly_allocation IS NULL OR monthly_allocation > 0),
    fund_day_of_month  SMALLINT NULL CHECK (fund_day_of_month IS NULL OR (fund_day_of_month >= 1 AND fund_day_of_month <= 31)),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_sg_user    FOREIGN KEY (user_id)    REFERENCES users(id),
    CONSTRAINT fk_sg_account FOREIGN KEY (account_id) REFERENCES accounts(id)
);

CREATE INDEX idx_sg_user_account ON saving_goals (user_id, account_id);

CREATE TRIGGER set_saving_goals_updated_at
    BEFORE UPDATE ON saving_goals
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_saving_goals_updated_at ON saving_goals;
DROP TABLE IF EXISTS saving_goals;
DROP TYPE IF EXISTS saving_goal_status;
-- +goose StatementEnd
