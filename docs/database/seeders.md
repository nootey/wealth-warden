# Seeding

Seeders populate the database with either initial production values or fake development/test data.

## Creating a Seeder

Seeders are handled manually. To create a new seeder, create it in `./databse/seeders/workers`
- Make sure to follow the proper declaration.
```go
func SeederName(ctx context.Context, db *gorm.DB) {
	
}
```

## Running seeders

Run the seeders with the following command:
```go 
go run ./cmd seed $(type)
```
You can use the following options:
- `basic` -> runs the basic seeders for a fresh rollout
- `full` -> runs all defined seeders, for faking data

<hr> 

Seeders require a .seeder.credentials file in `./pkg/config` to read values to seed.

Currently, these are the required parameters:

```js
SUPER_ADMIN_EMAIL=
SUPER_ADMIN_PASSWORD=
```