-- +goose Up
-- +goose StatementBegin
CREATE TABLE organization_features (
organization_id BIGINT UNSIGNED NOT NULL,
feature_id BIGINT UNSIGNED NOT NULL,
enabled BOOLEAN DEFAULT TRUE,
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
PRIMARY KEY (organization_id, feature_id),
CONSTRAINT fk_feature_org_organization FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
CONSTRAINT fk_feature_org_feature FOREIGN KEY (feature_id) REFERENCES features(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE organization_features;
-- +goose StatementEnd
