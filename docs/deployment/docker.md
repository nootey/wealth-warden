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
docker-compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up client -d
```
