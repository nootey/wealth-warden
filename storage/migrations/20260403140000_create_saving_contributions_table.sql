-- +goose Up
-- +goose StatementBegin
CREATE TYPE saving_contribution_source AS ENUM ('manual', 'auto');

CREATE TABLE saving_contributions (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    goal_id BIGINT NOT NULL,
    amount  NUMERIC(19,4) NOT NULL CHECK (amount > 0),
    month   DATE NOT NULL,
    note    VARCHAR(255) NULL,
    source  saving_contribution_source NOT NULL DEFAULT 'manual',

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_sc_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_sc_goal FOREIGN KEY (goal_id) REFERENCES saving_goals(id) ON DELETE CASCADE
);

CREATE INDEX idx_sc_goal_month ON saving_contributions (goal_id, month);

CREATE TRIGGER set_saving_contributions_updated_at
    BEFORE UPDATE ON saving_contributions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_saving_contributions_updated_at ON saving_contributions;
DROP TABLE IF EXISTS saving_contributions;
DROP TYPE IF EXISTS saving_contribution_source;
-- +goose StatementEnd
