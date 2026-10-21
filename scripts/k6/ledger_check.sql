-- Usage: psql -v run="k6 ledger <run id>%" < ledger_check.sql
-- Every row should be true in the last three columns
SELECT a.id,
       a.name,
       b.balance                                 AS stored,
       x.expected,
       s.end_balance                             AS latest_snapshot,
       b.balance = x.expected                    AS matches_ledger,
       b.balance >= -COALESCE(a.credit_limit, 0) AS within_limit,
       s.end_balance = b.balance                 AS matches_snapshot
FROM accounts a
JOIN balances b ON b.account_id = a.id
CROSS JOIN LATERAL (
    -- Same formula as expectedBalanceSum in internal/repositories/balance_repository.go
    SELECT COALESCE(SUM(CASE WHEN t.direction = 'expense' THEN -t.amount ELSE t.amount END), 0)::numeric(19,4) AS expected
    FROM transactions t
    WHERE t.account_id = a.id AND t.deleted_at IS NULL
) x
LEFT JOIN LATERAL (
    SELECT end_balance
    FROM balance_snapshots
    WHERE account_id = a.id
    ORDER BY as_of DESC
    LIMIT 1
) s ON true
WHERE a.name LIKE :'run'
ORDER BY a.id;
