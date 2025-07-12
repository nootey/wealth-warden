# Migrations

Migrations are handled using [Goose](https://github.com/pressly/goose).

## Creating a Migration

```sh
goose -dir ./pkg/database/migrations create create_{table}_table sql
```

This will generate a new .sql file where you define both Up and Down migration steps.

## Running Migrations

Use the following command to apply or rollback migrations:

```go 
go run ./cmd migrate $(type)
```

Options include:
- `up` -> runs all migrations
- `down` -> rolls back migrations
- `status` -> check status of migrations
- `fresh` -> drops database and runs all migrations
- `fresh-seed-basic` -> runs fresh migrations and basic seeders
- `fresh-seed-full` -> runs fresh migrations and all defined seeders
    - you can include/exclude seeders in `./databse/seeders/main.go`