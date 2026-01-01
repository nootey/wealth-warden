# Database

The app uses **PostgreSQL**.

### Local Setup

To spin up a local database instance:

```sh
docker-compose -f ./deployments/docker/docker-compose.yaml -p wealth-warden up db -d
```

This will run a Docker container running Postgres, and expose it on the configured port. You can connect via any Postgres-compatible client.

💡 Run this first before any migrations or seeders.

## Migrations

Migrations are handled using [Goose](https://github.com/pressly/goose).

### Creating a Migration

```sh
goose -dir ./storage/migrations create create_{table}_table sql
```

This will generate a new .sql file where you define both Up and Down migration steps.

### Running Migrations

Use the following command to apply or rollback migrations:

```go 
go run ./cmd migrate $(type) -d "/app/migrations"
```

Options include:
- `up` -> runs all migrations
- `down` -> rolls back migrations
- `status` -> check status of migrations
- `fresh` -> drops database and runs all migrations
- `fresh-seed-basic` -> runs fresh migrations and basic seeders
- `fresh-seed-full` -> runs fresh migrations and all defined seeders
    - you can include/exclude seeders in `./databse/seeders/main.go`

Available flags:
- `-d or --dir` -> specify the directory containing migration files (default is `./storage/migrations`)

## Seeding

Seeders populate the database with either initial production values or fake development/test data.

### Creating a Seeder

Seeders are handled manually. To create a new seeder, create it in `./databse/seeders/workers`
- Make sure to follow the proper declaration.
```go
func(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	
}
```

### Running seeders

Run the seeders with the following command:
```go 
go run ./cmd seed $(type)
```
You can use the following options:
- `basic` -> runs the basic seeders for a fresh rollout
- `full` -> runs all defined seeders, for faking data