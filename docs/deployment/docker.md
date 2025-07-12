## Docker setup

### Dockerfile

The app uses a Dockerfile, which is provided in `./build/Dockerfile`
- It's still a work in progress

### Docker compose

The app can be fully served with `docker-compose`.
- It is located in `./deployments/docker`


### Deployment

To spin up just the db component, you can use:

```sh
docker-compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up db -d
```

For the first time setup, you must run migrations!

```sh
docker-compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden run --rm app migrate fresh-seed-basic
```

To run the base image, which will run the `http` command, use:

```sh
docker-compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up app -d
```
