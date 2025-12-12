# Wealth Warden 👋

An open-source finance tracker focused on simplicity and usability, based on a personal Excel spreadsheet.

## Deployment

It's recommended to run the app with Docker. Instructions can be found [here](./docs/docker.md)

### Local development

If you want to add changes to the app, you can run it locally.
You need to setup:
- A Postgres instance
- Have Go installed
- Have npm installed

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

#### Running the client

Change into the correct directory:
```shell
cd ./client
```

Install packages with npm:
```shell
npm install
```

Run the client
```shell
npm run dev
```
