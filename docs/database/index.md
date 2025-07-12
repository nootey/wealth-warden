# 🗄Database

The system uses **TimescaleDB** (a PostgreSQL extension for time-series data). A docker-compose setup is available for local development.

### Local Setup

To spin up a local database instance:

```sh
docker-compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up db -d
```

This will run TimescaleDB and expose it on the configured port. You can connect via any Postgres-compatible client.

💡 Run this first before any migrations or seeders.