-- +goose Up
-- +goose StatementBegin
CREATE TYPE category_classification AS ENUM ('income', 'expense', 'savings', 'investment', 'adjustment', 'uncategorized');
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE categories (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT,
    name VARCHAR(100) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    classification category_classification NOT NULL DEFAULT 'expense',
    parent_id BIGINT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ NULL,

    CONSTRAINT fk_categories_user FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) REFERENCES categories(id),
    CONSTRAINT uq_categories_name_class UNIQUE (name, classification),
    CONSTRAINT no_self_reference CHECK (id IS DISTINCT FROM parent_id)
);

CREATE INDEX idx_categories_user_class ON categories(user_id, classification);

CREATE INDEX idx_categories_deleted_at
    ON categories(deleted_at)
    WHERE deleted_at IS NOT NULL;

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
