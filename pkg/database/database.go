package database

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"sync"
	"time"
	"wealth-warden/pkg/config"
)

var (
	postgresDB *gorm.DB
	once       sync.Once
)

func ConnectToPostgres(cfg *config.Config) (*gorm.DB, error) {
	return ConnectToDatabase(cfg, cfg.Postgres.Database)
}

func ConnectToMaintenance(cfg *config.Config) (*gorm.DB, error) {
	return ConnectToDatabase(cfg, "postgres")
}

func ConnectToDatabase(cfg *config.Config, targetDB string) (*gorm.DB, error) {

	var logLevel logger.LogLevel
	if cfg.Release {
		logLevel = logger.Silent
	} else {
		logLevel = logger.Info
	}

	host := cfg.Postgres.Host

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		host, cfg.Postgres.User, cfg.Postgres.Password, targetDB, cfg.Postgres.Port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		log.Printf("Failed to connect to database at %s: %v", host, err)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Failed to get raw DB instance: %v", err)
		return nil, err
	}

	// Set connection pooling and check connectivity.
	sqlDB.SetConnMaxLifetime(time.Minute * 5)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	if err := sqlDB.Ping(); err != nil {
		log.Printf("Could not ping database at %s: %v", host, err)
		return nil, err
	}

	log.Printf("Connected to database at %s", host)
	return db, nil
}

func DisconnectPostgres() error {
	if postgresDB == nil {
		return nil
	}
	sqlDB, err := postgresDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func PingPostgresDatabase() error {
	if postgresDB == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := postgresDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}

	return sqlDB.Ping()
}

func ConnectWithoutDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s port=%d sslmode=disable TimeZone=UTC dbname=postgres",
		cfg.Postgres.Host, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Port)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func EnsureDatabaseExists(cfg *config.Config) error {
	// Connect without specifying the target database.
	db, err := ConnectWithoutDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgresDB server: %w", err)
	}
	// Ensure the connection is closed after use.
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// Check if the target database exists by querying pg_database.
	var exists int
	checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM pg_database WHERE datname = '%s'", cfg.Postgres.Database)
	err = db.Raw(checkQuery).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("error checking if database exists: %w", err)
	}

	// If the database doesn't exist, create it.
	if exists == 0 {
		log.Printf("Database '%s' does not exist, creating it...", cfg.Postgres.Database)
		createQuery := fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8'", cfg.Postgres.Database)
		if err := db.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		log.Printf("Database '%s' created successfully", cfg.Postgres.Database)
	} else {
		log.Printf("Database '%s' already exists", cfg.Postgres.Database)
	}

	return nil
}

func DropAndRecreateDatabase(db *sql.DB, cfg *config.Config) error {
	dbName := cfg.Postgres.Database

	// Terminate all active connections to the target database.
	terminateQuery := fmt.Sprintf(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '%s' AND pid <> pg_backend_pid();",
		dbName,
	)
	if _, err := db.Exec(terminateQuery); err != nil {
		return fmt.Errorf("failed to terminate connections to database %s: %w", dbName, err)
	}

	// Drop the database if it exists.
	dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\";", dbName)
	if _, err := db.Exec(dropQuery); err != nil {
		return fmt.Errorf("failed to drop database %s: %w", dbName, err)
	}

	// Create the database.
	createQuery := fmt.Sprintf("CREATE DATABASE \"%s\";", dbName)
	if _, err := db.Exec(createQuery); err != nil {
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	return nil
}
