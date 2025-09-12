## Deployment

### Local

#### Running the server
You can run the server with the following command:
```shell
go run ./cmd/http-server
```

Alternatively, you can use the provided Makefile.
```shell
make run | migrate {type} | seed
```

### Environment
The app uses a `dev.yaml` file, located in `/pkg/config/override/`
- An example for docker is provided in `/pkg/config/dev.docker.yaml` which is an unsecure dummy config for docker builds.