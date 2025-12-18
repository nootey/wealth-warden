# Self-hosting with Docker

This guide will help you setup, update, and maintain your self-hosted application with Docker Compose. 
Docker Compose is the most popular and recommended way to self-host the app.

## Setup

Follow the guide below to get your app running.

### Install Docker

- Install Docker Engine by following the official guide
- Start the Docker service on your machine

### Configure your Docker Compose file and environment variables

By default, the app will run with defaults, and does not require any environment variables.

> ⚠️ **Warning:** This makes the app very unsecure, since it uses default credentials and can be easily exploited.

It is recommended to create an override config file in `/pkg/config/override/dev.yaml` and fill it out with your information.

If you're deploying with Traefik, you can also create a file in `/deployments/docker/.env`, to configure your domain and Traefik email.

Both files have examples provided in their respected directories.

### Run the app

To spin up just the db component, you can use:

```sh
docker compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up db -d
```

For the first time setup, you must run migrations!

```sh
docker compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden --profile migrate run --rm migrate migrate fresh-seed-basic
```

Alternate command
```sh
docker compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden run --rm server migrate fresh-seed-basic
```

To run the app, which will run all docker services, use:

```sh
docker compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up -d
```
