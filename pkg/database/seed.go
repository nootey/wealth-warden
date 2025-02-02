package database

import (
	"gorm.io/gorm"
	"log"
	"wealth-warden/internal/models"
	"wealth-warden/pkg/config"
	"wealth-warden/pkg/utils"
)

func RunEssentialSeeders(db *gorm.DB, cfg *config.Config) error {

	err := createSuperAdmin(db, cfg)
	if err != nil {
		return err
	}

	log.Println("Essential seeding completed successfully!")
	return nil
}

func createSuperAdmin(db *gorm.DB, cfg *config.Config) error {
	// Check if super-admin already exists to avoid duplicate creation
	var existingUser models.User
	if err := db.Where("role = ?", "super-admin").First(&existingUser).Error; err == nil {
		log.Println("Super-admin user already exists, skipping creation.")
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	hashedPassword, err := utils.HashAndSaltPassword(cfg.SuperAdminPassword)
	if err != nil {
		return err
	}

	superAdmin := models.User{
		Username: "admin",
		Email:    "support@wealth-warden.com",
		Password: hashedPassword,
		Role:     "super-admin",
	}

	if err := db.Create(&superAdmin).Error; err != nil {
		return err
	}

	log.Println("Super-admin user created successfully!")
	return nil
}
