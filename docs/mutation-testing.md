# Mutation tests

Mutation tests check how good the tests are. A tool makes a small change to the code, for example `>` to `>=`. Then it runs the tests.

- If a test fails, the change is **killed**. Good: the tests caught it.
- If all tests pass, the change **survived**. A test is missing an assertion or an edge case.

Coverage only shows that a line ran. Mutation tests show that a test checked the result.

These runs are slow, so they are local only. CI does not run them.

## Before you run

Install [Gremlins](https://gremlins.dev) for the Go side:

```sh
go install github.com/go-gremlins/gremlins/cmd/gremlins@v0.6.0
```

The client side uses [Stryker](https://stryker-mutator.io). It is a dev dependency, so `pnpm install` in `./client` installs it.

## Run

```sh
make test-mutation                       # Go and client, every package
make test-mutation target=go             # or target=client
make test-mutation only="utils finance"  # only packages whose path contains a filter
make test-mutation jobs=1                # packages at the same time (default 2)
make test-mutation integration=1         # include integration-tagged Go tests
```

The script runs one Gremlins or Stryker process per package, several at a time. Together they use half the CPUs, because every change also compiles in parallel.

- **Go**: every package with tests. The script copies the Go side of the repo to a temp folder first, so you can keep working while it runs.
- **Client**: every folder in `src` with tests. Only files with a sibling `.test.ts` file are changed, so a survivor points to a weak test, not to a file with no tests.

Each Go package also gets a memory cap: its share of half the RAM. Some changes make a loop endless, and the test then fills the memory. The cap stops only that test, and Gremlins counts the change as killed. The cap needs `systemd-run`. Without it, the script prints a warning and runs with no cap.

`integration=1` is very slow. Every single change runs the package tests again, and each run starts its own Postgres container. Use it with a filter, for example `only=internal/services`.

## Read the results

Each run writes to `logs/mutation/<timestamp>/`. `logs/mutation/latest` points to the last run.

| File | What it holds |
|---|---|
| `summary.txt` | One line per package: score, survived, killed, timed out, not covered |
| `go/<package>.log` | Gremlins output. Look for `LIVED` lines: `file.go:line:column` |
| `client/<folder>.log` | Stryker output. Each `[Survived]` block shows the changed code |

The score is the share of tested changes that the tests caught. It ignores code that no test runs.

Not every survivor is a bug. Some changes do not change the behavior, for example a changed error message that no test reads. Use the survivors as a list of places to look, not as a target to reach 100%.

A `TIMED OUT` result can come from a heavy machine. Run that package again with fewer `jobs`.

A `STOPPED` result means Gremlins quit before the end and wrote no summary. The log shows how far it got.
