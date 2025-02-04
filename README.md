# wealth-warden

Wealth warden is a finance tracking app, focused on observability. Its intention is to 
provide a cleaner interface to excel, while also providing visualization of your personal finances.

## Running the server
You can run the server with the following command:
```shell
go run cmd/http-server/main.go
```

Alternatively, you can use the provided Makefile.
```shell
make run | migrate {type} | seed
```


## Migrations

To create a migration, use this command: 
```sh
goose -dir .\pkg\database\migrations\ create create_{table}_table sql
```

You can use the following options:
- up -> runs all migrations
- down -> rolls back migrations
- status -> check status of migrations
- fresh -> redo all migrations

## Seeding

To create a seeder with goose, use this command:
```sh
goose -dir .\pkg\database\seeders\ create seed_{table}.go
```