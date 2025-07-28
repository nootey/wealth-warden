-- +goose Up
-- +goose StatementBegin
-- Create materialized view for monthly financial summary
CREATE MATERIALIZED VIEW monthly_summary AS
SELECT
    t.user_id,
    EXTRACT(YEAR FROM t.txn_date)::INT AS year,
        EXTRACT(MONTH FROM t.txn_date)::INT AS month,
        DATE_TRUNC('month', t.txn_date)::DATE AS period,

        SUM(CASE WHEN c.classification = 'income' THEN t.amount ELSE 0 END) AS inflow,
        SUM(CASE WHEN c.classification = 'expense' THEN t.amount ELSE 0 END) AS outflow,
        SUM(CASE WHEN c.classification = 'savings' THEN t.amount ELSE 0 END) AS savings,
        SUM(CASE WHEN c.classification = 'investment' THEN t.amount ELSE 0 END) AS investments,

        -- Net income = inflow - outflow
        SUM(CASE WHEN c.classification = 'income' THEN t.amount ELSE 0 END) -
        SUM(CASE WHEN c.classification = 'expense' THEN t.amount ELSE 0 END) AS net_income

FROM transactions t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.transaction_type = 'increase'
GROUP BY t.user_id, EXTRACT(YEAR FROM t.txn_date), EXTRACT(MONTH FROM t.txn_date), DATE_TRUNC('month', t.txn_date);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS monthly_summary CASCADE;
-- +goose StatementEnd
