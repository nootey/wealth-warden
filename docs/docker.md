# Self-hosting with Docker

This guide will help you setup, update, and maintain your self-hosted application with Docker Compose. 
Docker Compose is the recommended way to self-host the app.

## Setup

Follow the guide below to get your app running.

### Install Docker

- Install Docker Engine by following the guide
- Start the Docker service on your machine

### Configure your Docker Compose file and environment variables

By default, the app will run and does not require any environment variables.

> ⚠️ **Warning:** This makes the app very unsecure, since it uses default credentials and can be easily exploited.

It is recommended to create an override config file in `./config/dev.yaml` and fill it out with your information.

If you're deploying with Traefik, you can also create a file in `.env`, to configure your domain and Traefik email.

Both files have examples provided in their respected directories.

### Images

`docker-compose.yaml` pulls the published images, `nootey/wealth-warden` and `nootey/wealth-warden-client`.
Each release builds them for amd64 and arm64. `docker-compose.build.yaml` builds them from source instead.

Each `make docker-*` command pulls the latest published images, so `make docker-up` also updates the app.
To build from source instead, add `build=1`, for example `make docker-up build=1`.

### Migrations

The `migrate` service runs `migrate up` before the server starts.
Each start applies new migrations, so the first start also creates the schema.

## Run on a server

Traefik gets a TLS certificate from Let's Encrypt.
Your domain (`HOST` in `.env`) must point to the server, and ports 80 and 443 must be open.

To start all services, including the observability stack, use:

```sh
make docker-up
```

The app is then available at `https://<HOST>`.

To restart or stop all services, use:

```sh
make docker-restart
make docker-down
```

Without make, the same stack is:

```sh
docker compose -f ./docker-compose.observability.yaml -f ./docker-compose.yaml up -d
```

To build from source instead, add the build overlay:

```sh
docker compose -f ./docker-compose.observability.yaml -f ./docker-compose.yaml -f ./docker-compose.build.yaml up -d --build
```

To spin up just the db component, you can use:

```sh
docker compose -f ./docker-compose.yaml up db -d
```

## Run on an ARM board

For a Raspberry Pi or another ARM board. The board runs no Traefik.
The client serves the app on `WEB_CLIENT_PORT` (default `5000`), and Grafana runs on `GRAFANA_PORT` (default `3000`).
Set both in `.env` if you need other ports.

Add `arm=1` to each `make docker-*` command:

```sh
make docker-up arm=1
```
To restart or stop all services, use:

```sh
make docker-restart arm=1
make docker-down arm=1
```

## Run behind your own reverse proxy

For a host that already runs a reverse proxy, such as Caddy, on a Docker network. The app then publishes no host ports and runs no Traefik.
The client and Grafana join the network set in `PROXY_NETWORK`, and the proxy reaches them by container name.

Set `PROXY_NETWORK` in `.env` to the name of the proxy's network. The network must already exist.
Then add `proxy=1` to the up, down and restart commands. It works with or without `arm=1`:

```sh
make docker-up arm=1 proxy=1
```

Point the proxy at `wealth-warden-client-1:80` for the app and `wealth-warden-grafana-1:3000` for Grafana.

## Seed the database

The seed data is generated in Go code. There are no SQL seed files. Both flows below use
the `migrate` service, because that service holds the app binary.

To drop the database, run migrations, and seed in one step, override the migrate command:

```sh
# Types: fresh-seed-basic (minimal), fresh-seed-full (more data)
docker compose -f ./docker-compose.yaml run --rm migrate migrate fresh-seed-basic
```

To seed a database that already has the schema, run the `seed` subcommand:

```sh
# Types: basic, full, bulk, individual
docker compose -f ./docker-compose.yaml run --rm migrate seed basic
```

The Makefile wraps both flows. On an ARM board, add `arm=1`:

```sh
make docker-migrate-fresh                   # drop + migrate + basic seed
make docker-migrate-fresh seed=full         # drop + migrate + full seed
make docker-seed                            # basic seed on an existing schema
make docker-seed-bulk users=1000            # bulk seed with N users

make docker-migrate-fresh arm=1
```

Seed credentials come from the `seed:` block in the config file (see `config/dev.example.yaml`).

## Observability

The observability stack (Prometheus, Tempo, Loki, Grafana) is included by default.
On a server, Grafana is available at `https://grafana.<HOST>`.
On an ARM board, Grafana is available at `http://<board-address>:<GRAFANA_PORT>`.
Make sure to set `GRAFANA_PASSWORD` in your `.env` file before deploying publicly.
