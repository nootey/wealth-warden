-- +goose Up
-- +goose StatementBegin
-- The amount an account started with lived in start_balance on its earliest
-- balance row. This turns it into a transaction, so the ledger is the whole record.

-- Closed accounts get an opening row too, and the trigger blocks posting to them.
-- Goose wraps this in a transaction, so a failure puts the trigger back.
ALTER TABLE transactions DISABLE TRIGGER trg_txn_prevent_post_to_closed;

-- The opening row is user editable, and the edit form needs a category on it.
WITH opening AS (
    SELECT DISTINCT ON (b.account_id)
           b.account_id, b.as_of, b.start_balance, b.currency, a.user_id
    FROM balances b
    JOIN accounts a ON a.id = b.account_id
    ORDER BY b.account_id, b.as_of
)
INSERT INTO transactions
    (user_id, account_id, category_id, direction, amount, currency, txn_date, description, transaction_type)
SELECT o.user_id,
       o.account_id,
       (SELECT c.id FROM categories c
         WHERE c.classification = 'uncategorized'
           AND c.deleted_at IS NULL
           AND (c.user_id IS NULL OR c.user_id = o.user_id)
         ORDER BY c.name
         LIMIT 1),
       (CASE WHEN o.start_balance < 0 THEN 'expense' ELSE 'income' END)::transaction_direction_enum,
       ABS(o.start_balance),
       o.currency,
       o.as_of,
       'Opening balance',
       'opening'
FROM opening o;

-- end_balance is generated as start_balance + cash_inflows - cash_outflows, so
-- moving the seed into the day's flow leaves every derived number untouched.
WITH opening AS (
    SELECT DISTINCT ON (account_id) id, start_balance
    FROM balances
    ORDER BY account_id, as_of
)
UPDATE balances b
SET cash_inflows  = b.cash_inflows  + GREATEST(o.start_balance, 0),
    cash_outflows = b.cash_outflows + GREATEST(-o.start_balance, 0),
    start_balance = 0,
    updated_at    = NOW()
FROM opening o
WHERE b.id = o.id;

-- Every rebuild now starts at accounts.opened_at, so a day holding an opening
-- transaction must never sit before it.
WITH opening AS (
    SELECT DISTINCT ON (account_id) account_id, as_of
    FROM balances
    ORDER BY account_id, as_of
)
UPDATE accounts a
SET opened_at = o.as_of::timestamp AT TIME ZONE 'UTC'
FROM opening o
WHERE a.id = o.account_id
  AND a.opened_at > o.as_of::timestamp AT TIME ZONE 'UTC';

ALTER TABLE transactions ENABLE TRIGGER trg_txn_prevent_post_to_closed;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- opened_at is not restored: the day it was moved to is the day the account's
-- history actually starts.
WITH opening AS (
    SELECT DISTINCT ON (account_id) account_id, id AS balance_id
    FROM balances
    ORDER BY account_id, as_of
),
amounts AS (
    SELECT account_id,
           SUM(CASE WHEN direction = 'expense' THEN -amount ELSE amount END) AS seed
    FROM transactions
    WHERE transaction_type = 'opening' AND deleted_at IS NULL
    GROUP BY account_id
)
UPDATE balances b
SET start_balance = amounts.seed,
    cash_inflows  = b.cash_inflows  - GREATEST(amounts.seed, 0),
    cash_outflows = b.cash_outflows - GREATEST(-amounts.seed, 0),
    updated_at    = NOW()
FROM opening o
JOIN amounts ON amounts.account_id = o.account_id
WHERE b.id = o.balance_id;

-- The delete trigger turns a DELETE into a soft delete unless this is set.
SET LOCAL ww.hard_delete = 'on';
DELETE FROM transactions WHERE transaction_type = 'opening';
-- +goose StatementEnd
