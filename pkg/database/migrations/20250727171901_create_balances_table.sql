-- +goose Up
-- +goose StatementBegin
CREATE TABLE balances (
id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
account_id BIGINT UNSIGNED NOT NULL,
as_of DATE NOT NULL,
start_balance     DECIMAL(19,4) NOT NULL DEFAULT 0,
cash_inflows      DECIMAL(19,4) NOT NULL DEFAULT 0,
cash_outflows     DECIMAL(19,4) NOT NULL DEFAULT 0,
non_cash_inflows  DECIMAL(19,4) NOT NULL DEFAULT 0,
non_cash_outflows DECIMAL(19,4) NOT NULL DEFAULT 0,
net_market_flows  DECIMAL(19,4) NOT NULL DEFAULT 0,  -- P/L or market movements
adjustments       DECIMAL(19,4) NOT NULL DEFAULT 0,  -- fees, corrections, interest
end_balance       DECIMAL(19,4) GENERATED ALWAYS AS (
                start_balance
                + cash_inflows - cash_outflows
                + non_cash_inflows - non_cash_outflows
                + net_market_flows
                + adjustments
                ) STORED,
currency CHAR(3) NOT NULL DEFAULT 'EUR',
created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
UNIQUE (account_id, as_of),
INDEX     idx_balances_account_asof     (account_id, as_of)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS balances;
-- +goose StatementEnd
