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
- Minimize comments unless essential for clarity

## Typography - ASCII Only
- Do not use em dashes. Use hyphens instead. 
- Do not use smart or curly quotes. Use straight quotes instead. 
- Do not use the ellipsis character. Use three plain dots instead. 
- Do not use Unicode bullets. Use hyphens or asterisks instead. 
- Do not use non-breaking spaces. 
- Do not modify content inside backticks. Treat it as a literal example.

## Accuracy and Speculation Control

- Ask clarifying questions when requirements are unclear
- Present options when multiple approaches exist
- Never speculate about code, files, or APIs you have not read.
- If unsure: say "I don't know." Never guess confidently.
- Do not create new files unless strictly necessary.

## Git

- Never commit unless explicitly requested
- Main branch for PRs: `development`

## Mocks

- Generated with `mockery`, live in `/mocks`
- Can be regenerated with `make mock`

## Development Guidelines

- Follow existing code patterns and conventions
- DO NOT suggest service to service injections, unless absolutely necessary
- For the client, do NOT write custom css classes, use inline primeflex classes where possible
- Consider security and performance implications
- Use make build and make lint to verify new code
- Test changes when possible