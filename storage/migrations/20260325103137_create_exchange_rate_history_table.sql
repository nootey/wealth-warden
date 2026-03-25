-- +goose Up
-- +goose StatementBegin
CREATE TABLE exchange_rate_history (
    from_currency  CHAR(3)        NOT NULL,
    to_currency    CHAR(3)        NOT NULL,
    as_of          DATE           NOT NULL,
    rate           NUMERIC(19, 6) NOT NULL,

    created_at     TIMESTAMPTZ    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMPTZ    DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (from_currency, to_currency, as_of)
);

CREATE INDEX idx_erh_lookup ON exchange_rate_history(from_currency, to_currency, as_of);

CREATE TRIGGER set_exchange_rate_history_updated_at
    BEFORE UPDATE ON exchange_rate_history
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_erh_lookup;
DROP TABLE IF EXISTS exchange_rate_history;
-- +goose StatementEnd
