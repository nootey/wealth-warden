-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS dynamic_category_mappings (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
dynamic_category_id BIGINT UNSIGNED NOT NULL,
related_type ENUM('inflow', 'outflow', 'dynamic') NOT NULL,
related_id BIGINT UNSIGNED NOT NULL,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
FOREIGN KEY (dynamic_category_id) REFERENCES dynamic_categories(id) ON DELETE CASCADE,
INDEX idx_dynamic_category (dynamic_category_id),
INDEX idx_related (related_id, related_type)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS dynamic_category_mappings;
-- +goose StatementEnd
