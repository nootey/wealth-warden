#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# --- Kill any running app instance ---
_kill_app() {
    local killed=0
    if pkill -f "go run ./cmd" 2>/dev/null; then killed=1; fi
    if pkill -f "build/wealthwarden" 2>/dev/null; then killed=1; fi
    # Also try by port (2000) as a fallback
    if command -v lsof &>/dev/null; then
        local pid
        pid=$(lsof -ti :2000 2>/dev/null || true)
        if [ -n "$pid" ]; then kill "$pid" 2>/dev/null && killed=1; fi
    fi
    if [ "$killed" -eq 1 ]; then
        echo "App stopped. Waiting for port to free..."
        sleep 2
    else
        echo "App not running."
    fi
}

# --- Find a PostgreSQL dump file in the root directory ---
DUMP_FILE=""
for f in "$ROOT_DIR"/*.sql; do
    [ -f "$f" ] || continue
    if head -5 "$f" | grep -q "PostgreSQL database dump"; then
        DUMP_FILE="$f"
        break
    fi
done

if [ -z "$DUMP_FILE" ]; then
    echo "Error: no PostgreSQL dump file found in $ROOT_DIR" >&2
    exit 1
fi

echo "Found dump: $DUMP_FILE"

_kill_app

# --- Resolve credentials (fall back to docker-compose defaults) ---
POSTGRES_USER="${POSTGRES_USER:-postgres}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-postgres}"
POSTGRES_DB="${POSTGRES_DB:-wealth_warden}"

ENV_FILE="$ROOT_DIR/.env"
if [ -f "$ENV_FILE" ]; then
    while IFS='=' read -r key value; do
        value="${value%%#*}"       # strip inline comments
        value="${value%"${value##*[! ]}"}"  # trim trailing whitespace
        case "$key" in
            POSTGRES_USER)     POSTGRES_USER="$value" ;;
            POSTGRES_PASSWORD) POSTGRES_PASSWORD="$value" ;;
            POSTGRES_DB)       POSTGRES_DB="$value" ;;
        esac
    done < <(grep -E "^POSTGRES_(USER|PASSWORD|DB)=" "$ENV_FILE")
fi

# --- Find the running Postgres container ---
CONTAINER_ID=$(docker ps --filter "ancestor=postgres" --format "{{.ID}}" | head -1)

if [ -z "$CONTAINER_ID" ]; then
    CONTAINER_ID=$(docker ps --filter "name=db" --format "{{.ID}}" | head -1)
fi

if [ -z "$CONTAINER_ID" ]; then
    CONTAINER_ID=$(docker ps --filter "name=postgres" --format "{{.ID}}" | head -1)
fi

if [ -z "$CONTAINER_ID" ]; then
    echo "Error: no running PostgreSQL container found" >&2
    exit 1
fi

CONTAINER_NAME=$(docker ps --filter "id=$CONTAINER_ID" --format "{{.Names}}")
echo "Container:  $CONTAINER_NAME ($CONTAINER_ID)"
echo "Database:   $POSTGRES_DB"
echo "User:       $POSTGRES_USER"
echo ""
read -rp "This will DROP and recreate '$POSTGRES_DB'. Continue? [y/N] " confirm
[[ "$confirm" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

_psql() {
    docker exec -i -e PGPASSWORD="$POSTGRES_PASSWORD" "$CONTAINER_ID" \
        psql -U "$POSTGRES_USER" "$@"
}

echo "Dropping database..."
_psql postgres -c "DROP DATABASE IF EXISTS \"$POSTGRES_DB\";"

echo "Creating database..."
_psql postgres -c "CREATE DATABASE \"$POSTGRES_DB\";"

echo "Restoring from $(basename "$DUMP_FILE") ..."
_psql "$POSTGRES_DB" < "$DUMP_FILE"

echo "Done. '$POSTGRES_DB' restored successfully."

# --- Migrate ---
echo ""
echo "Running migrations..."
make -C "$ROOT_DIR" migrate type=up

# --- Start app ---
echo ""
echo "Starting app (make run) ..."
make -C "$ROOT_DIR" run
