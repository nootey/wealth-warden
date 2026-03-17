-- +goose Up
-- +goose StatementBegin
CREATE TABLE asset_price_history (
 asset_id  BIGINT       NOT NULL,
 as_of     DATE         NOT NULL,
 price     NUMERIC(19,4) NOT NULL,
 currency  CHAR(3)      NOT NULL DEFAULT 'USD',

 updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

 PRIMARY KEY (asset_id, as_of),
 CONSTRAINT fk_aph_asset FOREIGN KEY (asset_id)
     REFERENCES investment_assets(id) ON DELETE CASCADE
);

CREATE INDEX idx_aph_asset_asof ON asset_price_history(asset_id, as_of);

CREATE TRIGGER set_asset_price_history_updated_at
    BEFORE UPDATE ON asset_price_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_aph_asset_asof;
DROP TABLE IF EXISTS asset_price_history;
-- +goose StatementEnd