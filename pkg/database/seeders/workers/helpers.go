package workers

import (
	"fmt"
	"gorm.io/gorm"
	"log"
	"os"
	"path/filepath"
	"strings"
	"wealth-warden/internal/models"
	"wealth-warden/internal/repositories"
)

func LoadSeederCredentials() (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error loading working directory: %v", err)
	}
	path := filepath.Join(wd, "pkg", "config", "configurable", ".seeder.credentials")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	creds := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		creds[key] = value
	}
	return creds, nil
}

func GetUser(db *gorm.DB) (*models.User, error) {
	creds, err := LoadSeederCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to load seeder credentials: %w", err)
	}
	email, ok := creds["SUPER_ADMIN_EMAIL"]
	if !ok || email == "" {
		return nil, fmt.Errorf("SUPER_ADMIN_EMAIL not set in seeder credentials")
	}
	userRepo := repositories.NewUserRepository(db)
	user, err := userRepo.FindUserByEmail(nil, email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	return user, nil
}
