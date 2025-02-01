package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Release        string
	ClientDomain   string
	ClientPort     string
	HttpServerPort string
	MySQLHost      string
	MySQLUser      string
	MySQLPassword  string
	MySQLPort      int
	MySQLDatabase  string
}

func LoadConfig() *Config {

	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mysqlPort, err := strconv.Atoi(os.Getenv("MYSQL_PORT"))
	if err != nil {
		log.Fatalf("Invalid MySQL port: %v", err) // Handle conversion error
	}

	return &Config{
		Release:        os.Getenv("RELEASE"),
		ClientDomain:   os.Getenv("CLIENT_DOMAIN"),
		ClientPort:     os.Getenv("CLIENT_PORT"),
		HttpServerPort: os.Getenv("HTTP_SERVER_PORT"),
		MySQLHost:      os.Getenv("MYSQL_HOST"),
		MySQLUser:      os.Getenv("MYSQL_USER"),
		MySQLPassword:  os.Getenv("MYSQL_PASSWORD"),
		MySQLPort:      mysqlPort,
		MySQLDatabase:  os.Getenv("MYSQL_DATABASE"),
	}
}
