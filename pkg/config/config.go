package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"strconv"
)

type Config struct {
	Release              bool
	WebClientDomain      string
	WebClientPort        string
	HttpServerPort       string
	MySQLHost            string
	MySQLUser            string
	MySQLPassword        string
	MySQLPort            int
	MySQLDatabase        string
	JwtWebClientAccess   string
	JwtWebClientRefresh  string
	JwtWebClientEncodeID string
	Host                 string
}

func LoadConfig() *Config {

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	mysqlPort, err := strconv.Atoi(os.Getenv("MYSQL_PORT"))
	if err != nil {
		log.Fatalf("Invalid MySQL port: %v", err)
	}

	release, err := strconv.ParseBool(os.Getenv("RELEASE"))
	if err != nil {
		log.Fatalf("Invalid release mode: %v", err)
	}

	host := os.Getenv("HOST")
	if host == "" {
		log.Println("Host not defined in environment variables, using global fallback ...")
		host = "0.0.0.0"
	}

	return &Config{
		Release:              release,
		WebClientDomain:      os.Getenv("WEB_CLIENT_DOMAIN"),
		WebClientPort:        os.Getenv("WEB_CLIENT_PORT"),
		HttpServerPort:       os.Getenv("HTTP_SERVER_PORT"),
		MySQLHost:            os.Getenv("MYSQL_HOST"),
		MySQLUser:            os.Getenv("MYSQL_USER"),
		MySQLPassword:        os.Getenv("MYSQL_PASSWORD"),
		MySQLPort:            mysqlPort,
		MySQLDatabase:        os.Getenv("MYSQL_DATABASE"),
		JwtWebClientAccess:   os.Getenv("JWT_WEB_CLIENT_ACCESS"),
		JwtWebClientRefresh:  os.Getenv("JWT_WEB_CLIENT_REFRESH"),
		JwtWebClientEncodeID: os.Getenv("JWT_WEB_CLIENT_ENCODE_ID"),
		Host:                 host,
	}
}
