# Wealth Warden - Claude instructions

## Project context

Ledger based personal finance manager with an included web based client (separate CLAUDE.md file in /client).

- Jobs: `internal/jobscheduler` for scheduled/recurring jobs, `internal/queue` for async HTTP-triggered jobs
- Exchange rates: `GetExchangeRate` with a date caches to `exchange_rate_history`; without a date it's a live rate and never cached

## Workflow

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- Wait for explicit approval before writing any code or changing files
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## Development Guidelines

- For exploration tasks (finding files, grepping), prefer spawning Explore subagents rather than reading into main context
- DO NOT suggest service to service injections, unless absolutely necessary - present your reasoning if so
- Match existing code patterns and conventions even if you'd do it differently
- Minimum code that solves the problem. Nothing speculative.
- Don't "improve" adjacent code, comments, or formatting

## General guidelines
- Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify
- Don't assume. Don't hide confusion. Surface tradeoffs
- Define success criteria. Loop until verified 
- Transform tasks into verifiable goals:
  - "Add validation" → "Write tests for invalid inputs, then make them pass"
  - "Fix the bug" → "Write a test that reproduces it, then make it pass"
  - "Refactor X" → "Ensure tests pass before and after"