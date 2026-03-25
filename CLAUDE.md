# Wealth Warden - Claude instructions

## Project context

Ledger based personal finance manager with an included web based client.

- Jobs: `internal/jobscheduler` for scheduled/recurring jobs, `internal/queue` for async HTTP-triggered jobs
- Exchange rates: `GetExchangeRate` with a date caches to `exchange_rate_history`; without a date it's a live rate and never cached

## Response Style

- Keep responses concise and actionable
- Focus on essential information
- Provide details when specifically requested

## Code Style

- Write clean, self-documenting code
- Use descriptive variable and function names
- Minimize comments unless essential for clarity

## Workflow

- Confirm understanding before proceeding
- Present options when multiple approaches exist

## Git

- Never commit unless explicitly requested
- Main branch for PRs: `development`

## Mocks

- Generated with `mockery`, live in `/mocks`
- Can be regenerated with `make mock`

## Development Guidelines

- Follow existing code patterns and conventions
- DO NOT suggest service to service injections, unless absolutely necessary
- Consider security and performance implications
- Use make build and make lint to verify new code
- Test changes when possible