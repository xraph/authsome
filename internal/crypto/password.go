package crypto

import (
	"golang.org/x/crypto/bcrypt"
)

const defaultCost = bcrypt.DefaultCost

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), defaultCost)

	return string(bytes), err
}

// CheckPassword checks if a password matches a hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err == nil
}
