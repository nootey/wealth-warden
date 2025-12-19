-- +goose Up
-- +goose StatementBegin
CREATE TABLE category_groups (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    classification VARCHAR(100) NOT NULL,
    description TEXT NULL,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_category_group_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_category_group_user ON category_groups(user_id);

CREATE TRIGGER set_category_group_updated_at
    BEFORE UPDATE ON category_groups
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE category_group_members (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_group_members_group FOREIGN KEY (group_id) REFERENCES category_groups(id) ON DELETE CASCADE,
    CONSTRAINT fk_group_members_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE,
    CONSTRAINT uq_group_category UNIQUE (group_id, category_id)
);

CREATE INDEX idx_group_members_group ON category_group_members(group_id);
CREATE INDEX idx_group_members_category ON category_group_members(category_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS category_group_members;
DROP TABLE IF EXISTS category_group;
-- +goose StatementEnd