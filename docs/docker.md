# Self-hosting with Docker

This guide will help you setup, update, and maintain your self-hosted application with Docker Compose. 
Docker Compose is the most popular and recommended way to self-host the app.

## Setup

Follow the guide below to get your app running.

### Install Docker

- Install Docker Engine by following the official guide
- Start the Docker service on your machine

### Configure your Docker Compose file and environment variables

By default, the app will run and does not require any environment variables.

> ⚠️ **Warning:** This makes the app very unsecure, since it uses default credentials and can be easily exploited.

It is recommended to create an override config file in `./config/dev.yaml` and fill it out with your information.

If you're deploying with Traefik, you can also create a file in `.env`, to configure your domain and Traefik email.

Both files have examples provided in their respected directories.

### Run the app

To spin up just the db component, you can use:

```sh
docker compose -f ./docker-compose.yaml up db -d
```

For the first time setup, you must run migrations!

```sh
# Run migrations. Without an argument the migrate service runs `migrate up`.
docker compose -f ./docker-compose.yaml run --rm migrate
```

#### Seed the database

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

The Makefile wraps both flows:

```sh
make docker-migrate type=fresh-seed-basic   # drop + migrate + seed
make docker-seed type=basic                 # seed an existing schema
make docker-seed type=bulk users=1000       # bulk seed with N users
```

Seed credentials come from the `seed:` block in the config file (see `config/dev.example.yaml`).

To run the app, which will run all docker services including the observability stack, use:

```sh
docker compose -f ./docker-compose.observability.yaml -f ./docker-compose.yaml up -d
```

`docker-compose.yaml` only pulls published images. To build from source instead, add the build overlay:

```sh
docker compose -f ./docker-compose.yaml -f ./docker-compose.build.yaml up -d --build
```

The observability stack (Prometheus, Tempo, Grafana) is included by default. 
Grafana is available at `https://grafana.<your-domain>`. 
Make sure to set `GRAFANA_PASSWORD` in your `.env` file before deploying publicly.
