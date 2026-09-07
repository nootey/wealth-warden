## Docker setup

### Dockerfile

The app uses a Dockerfile, which is provided in `./build/Dockerfile`

- It's still a work in progress

### Docker compose

The app can be fully served with `docker-compose`.

- The compose files are located in the repository root

### Deployment

To spin up just the db component, you can use:

```sh
docker-compose -f ./docker-compose.yaml -p wealth-warden up client -d
```
