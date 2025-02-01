package utils

import "golang.org/x/crypto/bcrypt"

func HashAndSaltPassword(password string) (string, error) {
	// Generate a hashed and salted password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}
