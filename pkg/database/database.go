package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	"wealth-warden/pkg/config"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	postgresDB *gorm.DB
)

func ConnectToPostgres(cfg *config.Config, zapLogger *zap.Logger) (*gorm.DB, error) {
	return ConnectToDatabase(cfg, cfg.Postgres.Database, zapLogger)
}

func ConnectToMaintenance(cfg *config.Config, zapLogger *zap.Logger) (*gorm.DB, error) {
	return ConnectToDatabase(cfg, "postgres", zapLogger)
}

func ConnectToDatabase(cfg *config.Config, targetDB string, zapLogger *zap.Logger) (*gorm.DB, error) {

	host := cfg.Postgres.Host

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		host, cfg.Postgres.User, cfg.Postgres.Password, targetDB, cfg.Postgres.Port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Silent,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
	})
	if err != nil {
		zapLogger.Error("Failed to connect to database",
			zap.String("host", host),
			zap.String("database", targetDB),
			zap.Error(err))
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		zapLogger.Error("Failed to get raw DB instance", zap.Error(err))
		return nil, err
	}

	// Set connection pooling and check connectivity.
	sqlDB.SetConnMaxLifetime(time.Minute * 5)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	if err := sqlDB.Ping(); err != nil {
		zapLogger.Error("Could not ping database",
			zap.String("host", host),
			zap.Error(err))
		return nil, err
	}

	zapLogger.Info("Connected to database",
		zap.String("host", host),
		zap.String("database", targetDB))
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

func EnsureDatabaseExists(cfg *config.Config, zapLogger *zap.Logger) error {
	// Connect without specifying the target database.
	db, err := ConnectWithoutDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgresDB server: %w", err)
	}
	// Ensure the connection is closed after use.
	sqlDB, _ := db.DB()
	defer func(sqlDB *sql.DB) {
		err = sqlDB.Close()
		if err != nil {
			zapLogger.Error("Failed to close DB connection", zap.Error(err))
		}
	}(sqlDB)

	// Check if the target database exists by querying pg_database.
	var exists int
	checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM pg_database WHERE datname = '%s'", cfg.Postgres.Database)
	err = db.Raw(checkQuery).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("error checking if database exists: %w", err)
	}

	// If the database doesn't exist, create it.
	if exists == 0 {
		zapLogger.Info("Database does not exist, creating it", zap.String("database", cfg.Postgres.Database))
		createQuery := fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8'", cfg.Postgres.Database)
		if err := db.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		zapLogger.Info("Database created successfully", zap.String("database", cfg.Postgres.Database))
	} else {
		zapLogger.Info("Database already exists", zap.String("database", cfg.Postgres.Database))
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
