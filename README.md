# Wealth Warden 👋

An open-source finance tracker focused on simplicity and usability, based on a personal Excel spreadsheet.

## 🚀 About Wealth Warden

Wealth Warden is a lightweight, ledger-based finance tracker inspired by a personal Excel workflow. 
It focuses on simplicity, transparency, and usability - no bloat and no unnecessary complexity. 
Just your finances, tracked the way you want.

Whether you're managing multiple accounts, reviewing your cash flow, or monitoring long-term trends, Wealth Warden helps you stay organized and aware.

## 🎯 Features

- Easy-to-use interface.
- Transaction based income and expense tracking.
- Data Visualization – Simple charts for quick insights.
- Custom Categories – Personalize your tracking system.
- Open Source – You can confirm your data is not being manipulated.
- Ability to self-host with Docker.

## 🛠️ Tech Stack
Server: Go + Gin
Client: Vue + Vite
Database: Postgres with GORM

## 📦 Deployment

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
