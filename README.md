# Wealth Warden 👋

An open-source finance tracker focused on simplicity and usability.

## 🚀 About Wealth Warden
Wealth Warden is a personal finance tracker designed to be simple, intuitive, and efficient. Inspired by my own Excel-based template, this project aims to provide a seamless experience for tracking income, expenses, and financial goals—without unnecessary complexity.

## 🎯 Features
- Easy-to-use interface – No clutter, just what you need.
- Income & Expense Tracking – Stay on top of your cash flow.
- Budgeting Tools – Set and manage your financial goals.
- Data Visualization – Simple charts for quick insights.
- Custom Categories – Personalize your tracking system.
- Open Source – You can confirm your data is not being manipulated.

## 🛠️ Tech Stack
Server: Go
Database: MySQL with GORM

## 📦 Deployment

### Running the server
You can run the server with the following command:
```shell
go run cmd/http-server/main.go
```

Alternatively, you can use the provided Makefile.
```shell
make run | migrate {type} | seed
```

### Migrations

To create a migration, use this command: 
```sh
goose -dir .\pkg\database\migrations\ create create_{table}_table sql
```

You can use the following options:
- up -> runs all migrations
- down -> rolls back migrations
- status -> check status of migrations
- fresh -> redo all migrations

### Seeding

To create a seeder with goose, use this command:
```sh
goose -dir .\pkg\database\seeders\ create seed_{table}.go
```