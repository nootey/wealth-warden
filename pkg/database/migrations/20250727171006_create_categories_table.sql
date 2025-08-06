-- +goose Up
-- +goose StatementBegin
CREATE TYPE category_classification AS ENUM ('income', 'expense', 'savings', 'investment');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT,
    name VARCHAR(100) NOT NULL,
    classification category_classification NOT NULL DEFAULT 'expense',
    parent_id BIGINT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_categories_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE SET NULL,
    CONSTRAINT uq_categories_user_name_class UNIQUE (user_id, name, classification),
    CONSTRAINT no_self_reference CHECK (id IS DISTINCT FROM parent_id)
);

CREATE INDEX idx_categories_user_class ON categories(user_id, classification);

CREATE TRIGGER set_categories_updated_at
    BEFORE UPDATE ON categories
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS set_categories_updated_at ON categories;
DROP TABLE IF EXISTS categories;
DROP TYPE IF EXISTS category_classification;
-- +goose StatementEnd
