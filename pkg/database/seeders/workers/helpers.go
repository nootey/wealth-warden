package workers

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

func LoadSeederCredentials() (map[string]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error loading secrets: %v", err)
	}
	path := filepath.Join(wd, "pkg", "config", ".seeder.credentials")
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
