# Wealth Warden - Claude instructions

## Project context

Ledger based personal finance manager with an included web based client (separate CLAUDE.md file).

- Jobs: `internal/jobscheduler` for scheduled/recurring jobs, `internal/queue` for async HTTP-triggered jobs
- Exchange rates: `GetExchangeRate` with a date caches to `exchange_rate_history`; without a date it's a live rate and never cached

## Workflow

- Wait for explicit approval before writing any code or changing files
- Ask clarifying questions when requirements are unclear
- Present options when multiple approaches exist
- Never speculate about code, files, or APIs you have not read.

## Development Guidelines

- For exploration tasks (finding files, grepping), prefer spawning Explore subagents rather than reading into main context
- Use make build to verify new code
- DO NOT suggest service to service injections, unless absolutely necessary - present your reasoning if so
- Follow existing code patterns and conventions
- Consider security and performance implications

## Mocks

- Generated with `mockery`, live in `/mocks`
- Can be regenerated with `make mock`

## Response Style

- Keep responses concise and actionable
- Focus on essential information
- Provide details when specifically requested

## Code Style

- Write clean, self-documenting code
- Minimize comments unless essential for clarity

## Git

- Never commit unless explicitly requested
- Main branch for PRs: `development`