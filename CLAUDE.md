# Wealth Warden - Claude instructions

## Project context

Ledger-based personal finance manager with an included web=based client (separate CLAUDE.md file in /client).

- Jobs: `internal/jobqueue` holds the contract (args, kinds, dispatcher) and is safe for services to import; `internal/jobs` holds the runtime (job code, workers, River client, periodic schedule)
- Exchange rates: `GetExchangeRate` with a date caches to `exchange_rate_history`; without a date it's a live rate and never cached
- *_models contain constants, DB models and schemas, for each domain

## Rules

- DO NOT suggest service to service injections, unless absolutely necessary - present your reasoning if so
- Match existing repository code patterns and conventions. If you'd do it differently, suggest
- Minimize helpers in any domain/service files. If they are needed, create them in utils package.
- DO NOT create separate test files, use shared per domain/service ones.