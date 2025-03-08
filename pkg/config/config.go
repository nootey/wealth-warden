package config

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/pbkdf2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Release            bool
	WebClientDomain    string
	WebClientPort      string
	HttpServerPort     string
	SuperAdminPassword string
	MySQLHost          string
	MySQLUser          string
	MySQLPassword      string
	MySQLPort          int
	MySQLDatabase      string
}

// loadSecrets reads the secrets from the .env.secret file.
func loadSecrets(secretsPath string) ([]byte, string, error) {
	file, err := os.Open(secretsPath)
	if err != nil {
		return nil, "", fmt.Errorf("secrets file does not exist: %w", err)
	}
	defer file.Close()

	var saltStr, storedPass string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "SALT=") {
			saltStr = strings.TrimPrefix(line, "SALT=")
		} else if strings.HasPrefix(line, "PASS=") {
			storedPass = strings.TrimPrefix(line, "PASS=")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", err
	}
	if saltStr == "" || storedPass == "" {
		return nil, "", errors.New("SALT or PASS not set in secrets file")
	}

	salt, err := base64.StdEncoding.DecodeString(saltStr)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode salt: %w", err)
	}
	return salt, storedPass, nil
}

// removePKCS7Padding removes PKCS7 padding from decrypted data.
func removePKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("decrypted data is empty")
	}
	padLen := int(data[len(data)-1])
	if padLen > aes.BlockSize || padLen == 0 {
		return nil, errors.New("invalid padding")
	}
	for i := 0; i < padLen; i++ {
		if data[len(data)-1-i] != byte(padLen) {
			return nil, errors.New("invalid padding")
		}
	}
	return data[:len(data)-padLen], nil
}

// DecryptEnvFile reads and decrypts the env file located at envPath using the provided secret.
func DecryptEnvFile(envPath string, secret string, salt []byte) (map[string]string, error) {
	content, err := ioutil.ReadFile(envPath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 {
		return nil, fmt.Errorf("env file does not have the expected format")
	}

	header := strings.TrimSpace(lines[0])
	if header != "ENCRYPTED" {
		return nil, fmt.Errorf("env file is not encrypted (header: %s)", header)
	}

	ivB64 := strings.TrimSpace(lines[1])
	ciphertextB64 := strings.TrimSpace(strings.Join(lines[2:], "\n"))
	iv, err := base64.StdEncoding.DecodeString(ivB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode IV: %v", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %v", err)
	}

	// Derive the key using PBKDF2 with the plain text password read from secrets.
	key := pbkdf2.Key([]byte(secret), salt, 100000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	decrypted := make([]byte, len(ciphertext))
	mode.CryptBlocks(decrypted, ciphertext)

	unpadded, err := removePKCS7Padding(decrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to remove padding: %v", err)
	}

	envMap, err := godotenv.Parse(bytes.NewReader(unpadded))
	if err != nil {
		return nil, fmt.Errorf("failed to parse decrypted env: %v", err)
	}
	return envMap, nil
}

func LoadConfig() *Config {

	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error loading secrets: %v", err)
	}
	secretsPath := filepath.Join(wd, "pkg", "config", ".env.secret")
	envPath := filepath.Join(wd, "pkg", "config", ".env")

	salt, storedPass, err := loadSecrets(secretsPath)
	if err != nil {
		log.Fatalf("Error loading secrets: %v", err)
	}

	envMap, err := DecryptEnvFile(envPath, storedPass, salt)
	if err != nil {
		log.Fatalf("Error decrypting env file: %v", err)
	}

	mysqlPort, err := strconv.Atoi(envMap["MYSQL_PORT"])
	if err != nil {
		log.Fatalf("Invalid MySQL port: %v", err)
	}

	release, err := strconv.ParseBool(envMap["RELEASE"])
	if err != nil {
		log.Fatalf("Invalid release mode: %v", err)
	}

	return &Config{
		Release:            release,
		WebClientDomain:    envMap["WEB_CLIENT_DOMAIN"],
		WebClientPort:      envMap["WEB_CLIENT_PORT"],
		HttpServerPort:     envMap["HTTP_SERVER_PORT"],
		SuperAdminPassword: envMap["SUPER_ADMIN_PASSWORD"],
		MySQLHost:          envMap["MYSQL_HOST"],
		MySQLUser:          envMap["MYSQL_USER"],
		MySQLPassword:      envMap["MYSQL_PASSWORD"],
		MySQLPort:          mysqlPort,
		MySQLDatabase:      envMap["MYSQL_DATABASE"],
	}
}
