-- +goose Up
-- +goose StatementBegin
CREATE TYPE category_necessity AS ENUM ('need', 'want');

ALTER TABLE categories
    ADD COLUMN necessity category_necessity NULL;

UPDATE categories SET necessity = 'want' WHERE classification = 'expense';

ALTER TABLE categories
    ADD CONSTRAINT chk_categories_necessity
    CHECK ((classification = 'expense') = (necessity IS NOT NULL));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE categories DROP COLUMN IF EXISTS necessity;
DROP TYPE IF EXISTS category_necessity;
-- +goose StatementEnd
