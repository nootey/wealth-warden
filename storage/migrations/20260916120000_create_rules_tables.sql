-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS rules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    match_type VARCHAR(10) NOT NULL DEFAULT 'all',
    effective_date DATE NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_rules_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_rules_user_id ON rules(user_id);

CREATE TRIGGER set_rules_updated_at
    BEFORE UPDATE ON rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TABLE IF NOT EXISTS rule_conditions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rule_id BIGINT NOT NULL,
    parent_id BIGINT NULL,
    is_group BOOLEAN NOT NULL DEFAULT FALSE,
    match_type VARCHAR(10) NOT NULL DEFAULT '',
    field VARCHAR(50) NOT NULL DEFAULT '',
    operator VARCHAR(20) NOT NULL DEFAULT '',
    value TEXT NOT NULL DEFAULT '',
    position INT NOT NULL DEFAULT 0,

    CONSTRAINT fk_rule_conditions_rule FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE,
    CONSTRAINT fk_rule_conditions_parent FOREIGN KEY (parent_id) REFERENCES rule_conditions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rule_conditions_rule_id ON rule_conditions(rule_id);

CREATE TABLE IF NOT EXISTS rule_actions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rule_id BIGINT NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    value TEXT NOT NULL DEFAULT '',
    position INT NOT NULL DEFAULT 0,

    CONSTRAINT fk_rule_actions_rule FOREIGN KEY (rule_id) REFERENCES rules(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rule_actions_rule_id ON rule_actions(rule_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS rule_actions;
DROP TABLE IF EXISTS rule_conditions;
DROP TRIGGER IF EXISTS set_rules_updated_at ON rules;
DROP TABLE IF EXISTS rules;
-- +goose StatementEnd
