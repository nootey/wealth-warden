package database

import (
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"sync"
	"time"
	"wealth-warden/pkg/config"
)

var (
	mysqlDB *gorm.DB
	once    sync.Once
)

// ConnectToMySQL initializes a singleton GORM connection to MySQL.
func ConnectToMySQL(cfg *config.Config, disableLogging bool) (*gorm.DB, error) {
	var err error

	once.Do(func() {
		hosts := []string{cfg.MySQLHost, "localhost", "mysql"}
		logLevel := logger.Info
		if disableLogging {
			logLevel = logger.Silent
		}

		for _, host := range hosts {
			dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				cfg.MySQLUser, cfg.MySQLPassword, host, cfg.MySQLPort, cfg.MySQLDatabase)

			mysqlDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
				Logger: logger.Default.LogMode(logLevel),
			})
			if err != nil {
				log.Printf("Failed to connect to MySQL at %s: %v", host, err)
				continue
			}

			sqlDB, err := mysqlDB.DB()
			if err != nil {
				log.Printf("Failed to get mysql database instance: %v", err)
				continue
			}

			// Ping the database to check if the connection is alive
			sqlDB.SetConnMaxLifetime(time.Minute * 5)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetMaxOpenConns(100)

			if err := sqlDB.Ping(); err != nil {
				log.Printf("Could not ping MySQL at %s: %v", host, err)
				continue
			}

			log.Printf("Connected to MySQL at %s", host)
			return
		}

		err = fmt.Errorf("failed to connect to any MySQL host")
	})

	return mysqlDB, err
}

// DisconnectMySQL closes the database connection.
func DisconnectMySQL() error {
	if mysqlDB == nil {
		return nil
	}
	sqlDB, err := mysqlDB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func ConnectWithoutDB(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQLUser, cfg.MySQLPassword, cfg.MySQLHost, cfg.MySQLPort)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func PingMysqlDatabase() error {
	if mysqlDB == nil {
		return fmt.Errorf("database connection is nil")
	}

	sqlDB, err := mysqlDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}

	return sqlDB.Ping()
}

// EnsureDatabaseExists checks if the database exists, and if not, it creates it.
func EnsureDatabaseExists(cfg *config.Config) error {
	// Connect to MySQL without specifying a database
	db, err := ConnectWithoutDB(cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to MySQL server: %w", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	// Check if the database exists
	var exists int
	checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = '%s'", cfg.MySQLDatabase)
	err = db.Raw(checkQuery).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("error checking if database exists: %w", err)
	}

	// If the database doesn't exist, create it
	if exists == 0 {
		log.Printf("Database '%s' does not exist, creating it...", cfg.MySQLDatabase)
		createQuery := fmt.Sprintf("CREATE DATABASE %s CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.MySQLDatabase)
		if err := db.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		log.Printf("Database '%s' created successfully", cfg.MySQLDatabase)
	} else {
		log.Printf("Database '%s' already exists", cfg.MySQLDatabase)
	}

	return nil
}

func DropAndRecreateDatabase(db *sql.DB, cfg *config.Config) error {
	// Drop the database
	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", cfg.MySQLDatabase)); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	// Recreate the database
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE `%s`;", cfg.MySQLDatabase)); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Re-select the newly created database.
	if _, err := db.Exec(fmt.Sprintf("USE `%s`;", cfg.MySQLDatabase)); err != nil {
		return fmt.Errorf("failed to select database: %w", err)
	}

	return nil
}
