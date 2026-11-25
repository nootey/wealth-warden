#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
CLIENT_DIR="$ROOT_DIR/client"
LOG_ROOT="$ROOT_DIR/logs/mutation"
GREMLINS_VERSION="v0.6.0"

usage() {
    cat <<EOF
Usage: scripts/mutation-test.sh [options] [filter...]

Runs mutation tests package by package and writes one log per package.
Go packages use Gremlins. Client folders use Stryker.

Options:
  -t, --target go|client|all   what to test (default: all)
  -j, --jobs N                 packages to run at the same time (default: 2)
  -i, --integration            include integration-tagged Go tests (slow)
  -h, --help                   show this help

A filter keeps only the packages whose path contains it.
For example, "utils" runs pkg/utils and client src/utils.

Logs go to logs/mutation/<timestamp>/. logs/mutation/latest points to the last run.
EOF
}

TARGET="all"
JOBS=2
INTEGRATION=0
FILTERS=()

while [ $# -gt 0 ]; do
    case "$1" in
        -t|--target)      TARGET="${2:?--target needs a value}"; shift 2 ;;
        -j|--jobs)        JOBS="${2:?--jobs needs a value}"; shift 2 ;;
        -i|--integration) INTEGRATION=1; shift ;;
        -h|--help)        usage; exit 0 ;;
        -*)               echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
        *)                FILTERS+=("$1"); shift ;;
    esac
done

case "$TARGET" in
    go|client|all) ;;
    *) echo "Error: --target must be go, client or all" >&2; exit 1 ;;
esac

if ! [[ "$JOBS" =~ ^[1-9][0-9]*$ ]]; then
    echo "Error: --jobs must be a positive number" >&2
    exit 1
fi

RUN_GO=0; RUN_CLIENT=0
[ "$TARGET" != "client" ] && RUN_GO=1
[ "$TARGET" != "go" ] && RUN_CLIENT=1

if [ "$RUN_GO" -eq 1 ] && ! command -v gremlins &>/dev/null; then
    echo "Error: gremlins is not installed. Install it with:" >&2
    echo "  go install github.com/go-gremlins/gremlins/cmd/gremlins@$GREMLINS_VERSION" >&2
    exit 1
fi

if [ "$RUN_CLIENT" -eq 1 ] && [ ! -x "$CLIENT_DIR/node_modules/.bin/stryker" ]; then
    echo "Error: Stryker is not installed. Run 'pnpm install' in ./client" >&2
    exit 1
fi

# Use half the CPUs: every mutant also compiles in parallel, and a full machine makes mutants time out
WORKERS=$(( $(nproc) / (2 * JOBS) ))
if [ "$WORKERS" -lt 1 ]; then WORKERS=1; fi

# A mutant can make a loop endless and fill the RAM before the Gremlins timeout, so cap each package
GO_MEM_CAP=()
if [ "$RUN_GO" -eq 1 ]; then
    MEM_CAP_MB=$(( $(awk '/^MemTotal:/ {print $2}' /proc/meminfo) / 1024 / (2 * JOBS) ))
    MEM_CAP=(systemd-run --user --scope --quiet -p "MemoryMax=${MEM_CAP_MB}M" -p MemorySwapMax=0)
    if command -v systemd-run &>/dev/null && "${MEM_CAP[@]}" true &>/dev/null; then
        GO_MEM_CAP=("${MEM_CAP[@]}")
    else
        echo "Warning: systemd-run is not available, Go packages run without a memory cap" >&2
    fi
fi

STAMP="$(date +%Y%m%d-%H%M%S)"
RUN_DIR="$LOG_ROOT/$STAMP"
SUMMARY="$RUN_DIR/summary.txt"
mkdir -p "$RUN_DIR"
ln -sfn "$STAMP" "$LOG_ROOT/latest"

WORK_DIR="$(mktemp -d -t ww-mutation-XXXXXX)"

cleanup() {
    rm -rf "$WORK_DIR" "$CLIENT_DIR/.stryker-tmp"
}
trap cleanup EXIT
# Background jobs ignore Ctrl+C in a script, so stop the whole process group by hand
trap 'cleanup; trap - INT TERM EXIT; kill 0' INT TERM

matches_filter() {
    [ ${#FILTERS[@]} -eq 0 ] && return 0
    local f
    for f in "${FILTERS[@]}"; do
        [[ "$1" == *"$f"* ]] && return 0
    done
    return 1
}

run_pool() {
    local worker="$1"; shift
    local item
    for item in "$@"; do
        while [ "$(jobs -rp | wc -l)" -ge "$JOBS" ]; do wait -n || true; done
        "$worker" "$item" &
    done
    wait
}

record() {
    local kind="$1" name="$2" result="$3" elapsed="$4"
    local line
    line="$(printf '%-7s %-36s %s' "$kind" "$name" "$result")"
    echo "$line" >> "$SUMMARY"
    printf 'done   %s (%dm%02ds)\n' "$line" $(( elapsed / 60 )) $(( elapsed % 60 ))
}

go_summary() {
    awk '
        /^Killed:/           { gsub(",", ""); killed = $2; lived = $4; uncovered = $7 }
        /^Timed out:/        { gsub(",", ""); timedout = $3 }
        /^Test efficacy:/    { score = $3 }
        END {
            if (killed == "") exit 1
            if (killed + lived == 0) score = "-"
            printf "score %7s | survived %4s | killed %4s | timed out %3s | not covered %4s", score, lived, killed, timedout, uncovered
        }
    ' "$1"
}

client_summary() {
    awk -F'|' '
        /^All files/ {
            gsub(/ /, "")
            score = ($4 + $5 + $6 == 0) ? "-" : $3 "%"
            printf "score %7s | survived %4s | killed %4s | timed out %3s | not covered %4s", score, $6, $4, $5, $7
            found = 1
        }
        END { exit !found }
    ' "$1"
}

run_go_pkg() {
    local pkg="$1" start=$SECONDS status=0 result
    local log="$RUN_DIR/go/${pkg//\//_}.log"
    echo "start  go      $pkg"
    (cd "$WORK_DIR/src" && TMPDIR="$WORK_DIR/tmp" "${GO_MEM_CAP[@]}" gremlins unleash --workers "$WORKERS" "${GO_TAGS[@]}" "./$pkg") >"$log" 2>&1 || status=$?
    if [ "$status" -ne 0 ]; then
        result="FAILED (exit $status), see log"
    elif ! result="$(go_summary "$log")"; then
        result="no mutants"
    fi
    record go "$pkg" "$result" $(( SECONDS - start ))
}

run_client_dir() {
    local dir="$1" start=$SECONDS status=0 result
    local name="${dir//\//_}"
    local log="$RUN_DIR/client/$name.log"
    echo "start  client  $dir"
    (cd "$CLIENT_DIR" && pnpm exec stryker run \
        --mutate "${CLIENT_MUTATE[$dir]}" \
        --concurrency "$WORKERS" \
        --tempDirName ".stryker-tmp/$name") 2>&1 | sed -u 's/\x1b\[[0-9;]*m//g' >"$log" || status=$?
    if [ "$status" -ne 0 ]; then
        result="FAILED (exit $status), see log"
    elif ! result="$(client_summary "$log")"; then
        result="no mutants"
    fi
    record client "$dir" "$result" $(( SECONDS - start ))
}

commit="$(git -C "$ROOT_DIR" rev-parse --short HEAD)"
[ -n "$(git -C "$ROOT_DIR" status --porcelain)" ] && commit="$commit (uncommitted changes)"
{
    echo "Run:     $STAMP"
    echo "Commit:  $commit"
    echo "Options: target=$TARGET jobs=$JOBS workers=$WORKERS integration=$INTEGRATION filters=${FILTERS[*]:-none}"
    echo ""
} > "$SUMMARY"

echo "Logs: $RUN_DIR"
echo "Running $JOBS package(s) at a time, $WORKERS worker(s) each"
echo ""

if [ "$RUN_GO" -eq 1 ]; then
    GO_TAGS=()
    [ "$INTEGRATION" -eq 1 ] && GO_TAGS=(--tags integration)

    # Gremlins copies the whole module once per worker, so give it only the Go side of the repo
    mkdir -p "$WORK_DIR/src" "$WORK_DIR/tmp" "$RUN_DIR/go"
    git -C "$ROOT_DIR" ls-files -z --cached --others --exclude-standard -- ':!:client' ':!:docs' \
        | (cd "$ROOT_DIR" && tar --null --ignore-failed-read -T - -cf - 2>/dev/null) \
        | tar -xf - -C "$WORK_DIR/src"

    GO_PKGS=()
    while IFS= read -r dir; do
        pkg="${dir#"$WORK_DIR/src/"}"
        matches_filter "$pkg" && GO_PKGS+=("$pkg")
    done < <(cd "$WORK_DIR/src" && go list "${GO_TAGS[@]}" -f '{{if or .TestGoFiles .XTestGoFiles}}{{.Dir}}{{end}}' ./...)

    if [ ${#GO_PKGS[@]} -gt 0 ]; then
        run_pool run_go_pkg "${GO_PKGS[@]}"
    else
        echo "No Go packages match"
    fi
fi

if [ "$RUN_CLIENT" -eq 1 ]; then
    mkdir -p "$RUN_DIR/client"

    # Only files with a sibling test are mutated, so survivors point at weak tests, not missing ones
    declare -A CLIENT_MUTATE=()
    while IFS= read -r test_file; do
        src="${test_file%.test.ts}.ts"
        [ -f "$CLIENT_DIR/$src" ] || continue
        dir="$(dirname "$src")"
        matches_filter "$dir" || continue
        CLIENT_MUTATE[$dir]+="${CLIENT_MUTATE[$dir]:+,}$src"
    done < <(cd "$CLIENT_DIR" && find src -name '*.test.ts' | sort)

    if [ ${#CLIENT_MUTATE[@]} -gt 0 ]; then
        mapfile -t CLIENT_DIRS < <(printf '%s\n' "${!CLIENT_MUTATE[@]}" | sort)
        run_pool run_client_dir "${CLIENT_DIRS[@]}"
    else
        echo "No client folders match"
    fi
fi

{ head -n 4 "$SUMMARY"; tail -n +5 "$SUMMARY" | sort; } > "$SUMMARY.tmp" && mv "$SUMMARY.tmp" "$SUMMARY"

echo ""
cat "$SUMMARY"
