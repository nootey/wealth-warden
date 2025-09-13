-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS invitations (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username VARCHAR(128),
    display_name VARCHAR(192) NOT NULL,
    email VARCHAR(255) NOT NULL,
    hash VARCHAR(255) NOT NULL,
    role_id BIGINT NOT NULL,
    UNIQUE(email, role_id),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX invitations_username_role_id_unique
    ON invitations (username, role_id)
    WHERE username IS NOT NULL;

CREATE TRIGGER set_invitations_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS invitations;
-- +goose StatementEnd