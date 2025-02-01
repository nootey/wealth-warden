package migrations

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/database"
)

func RunBaseMigrations() error {
	cfg := config.LoadConfig()

	tempDb, err := database.ConnectWithoutDB(cfg)
	if err != nil {
		return err
	}

	dbName := cfg.MySQLDatabase

	// Ensure database exists
	if err := ensureDatabaseExists(tempDb, dbName); err != nil {
		return err
	}

	// Reconnect to the specific database
	db, err := database.ConnectToMySQL(cfg)
	if err != nil {
		return err
	}

	// Reset the database (drop all tables)
	if err := resetDatabase(db, dbName); err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	// Run migrations
	err = db.AutoMigrate(
		&models.User{},
	)
	if err != nil {
		return err
	}

	log.Println("Base database migrations completed successfully!")
	return nil
}

func ensureDatabaseExists(db *gorm.DB, dbName string) error {
	// Create the database if it doesn't exist
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", dbName)
	return db.Exec(query).Error
}

func resetDatabase(db *gorm.DB, dbName string) error {
	log.Println("Dropping all tables in database:", dbName)

	// Disable foreign key checks to prevent constraint issues
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 0").Error; err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %w", err)
	}

	// Drop all tables
	var tables []string
	if err := db.Raw("SELECT table_name FROM information_schema.tables WHERE table_schema = ?", dbName).Scan(&tables).Error; err != nil {
		return fmt.Errorf("failed to fetch table names: %w", err)
	}

	for _, table := range tables {
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)).Error; err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Re-enable foreign key checks
	if err := db.Exec("SET FOREIGN_KEY_CHECKS = 1").Error; err != nil {
		return fmt.Errorf("failed to re-enable foreign key checks: %w", err)
	}

	log.Println("Successfully reset the database by dropping all tables.")
	return nil
}
