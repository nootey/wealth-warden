## Docker setup

### Dockerfile

The app is built with a multi-stage Dockerfile at `./build/Dockerfile`.
- The image bakes in a default config: `/pkg/config/dev.docker.yaml` -> copied to `/app/pkg/config/dev.yaml`.
- You can override the config with: `/pkg/config/override/dev.yaml`
- Entry command is server http.

### Docker compose

The app can be fully served with `docker-compose`.
- It is located in `./deployments/docker`
- DB defaults: you can run without a .env. Defaults are provided.

### Deployment

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
