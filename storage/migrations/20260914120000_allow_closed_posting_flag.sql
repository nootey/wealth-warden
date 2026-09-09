-- +goose Up
-- +goose StatementBegin
-- Rebuilds stand the closed-account guard down while they repost a user's trade
-- cash. They used to do it with ALTER TABLE ... DISABLE TRIGGER, which locks the
-- whole transactions table against every other session. This session flag matches
-- ww.hard_delete and locks nothing.
CREATE OR REPLACE FUNCTION prevent_txn_on_closed_or_deleted_account()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    acc_closed_at TIMESTAMPTZ;
    acc_is_active  BOOLEAN;
BEGIN
IF current_setting('ww.allow_closed_posting', true) = 'on' THEN
    RETURN NEW;
END IF;

SELECT closed_at, is_active
INTO acc_closed_at, acc_is_active
FROM accounts
WHERE id = NEW.account_id;

IF acc_closed_at IS NOT NULL OR acc_is_active IS FALSE THEN
        RAISE EXCEPTION 'Account % is not open (inactive or deleted); cannot post transactions', NEW.account_id
            USING ERRCODE = 'check_violation';
END IF;

RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_txn_on_closed_or_deleted_account()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    acc_closed_at TIMESTAMPTZ;
    acc_is_active  BOOLEAN;
BEGIN
SELECT closed_at, is_active
INTO acc_closed_at, acc_is_active
FROM accounts
WHERE id = NEW.account_id;

IF acc_closed_at IS NOT NULL OR acc_is_active IS FALSE THEN
        RAISE EXCEPTION 'Account % is not open (inactive or deleted); cannot post transactions', NEW.account_id
            USING ERRCODE = 'check_violation';
END IF;

RETURN NEW;
END;
$$;
-- +goose StatementEnd
