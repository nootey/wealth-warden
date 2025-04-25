## Database

- Have an instance of mysql running.
- A docker-compose file is provided in `build/docker-compose/mysql`.
- Run the docker compose and create an image of the db.

### Migrations

To create a migration, use this command:
```sh
goose -dir .\pkg\database\migrations\ create create_{table}_table sql
```

Run the migrations with the following command:
```go 
go run ./cmd/http-server migrate {type}
```

You can use the following options:
- up -> runs all migrations
- down -> rolls back migrations
- status -> check status of migrations
- fresh -> drops database and runs all migrations
- fresh-seed-basic -> runs fresh migrations and basic seeders
- fresh-seed-full -> runs fresh migrations and all defined seeders
    - you can include/exclude seeders in `./databse/seeders/main.go`

## Seeding

Seeders are handled manually. To create a new seeder, create it in `./databse/seeders/workers`
- Make sure to follow the proper declaration.
```go
func SeederName(ctx context.Context, db *gorm.DB) {
	
}
```

Run the seeders with the following command:
```go 
go run ./cmd/http-server seed {type}
```
You can use the following options:
- basic -> runs the basic seeders for a fresh rollout
- full -> runs all defined seeders, for faking data

<hr> 

Seeders require a .seeder.credentials file in `./pkg/config` to read values to seed.

Currently, these are the required parameters:

```js
SUPER_ADMIN_EMAIL=
SUPER_ADMIN_PASSWORD= 
MEMBER_EMAIL=
MEMBER_PASSWORD=
```