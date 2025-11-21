package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomBytes generates n random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
	bytes := make([]byte, n)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// GenerateRandomString generates a random string of length n
func GenerateRandomString(n int) (string, error) {
	bytes, err := GenerateRandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// ValidatePassword validates password against security requirements
func ValidatePassword(password string) error {
	// Check minimum length
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters long")
	}

	// Check for uppercase, lowercase, number, and symbol
	var hasUpper, hasLower, hasNumber, hasSymbol bool
	for _, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			hasUpper = true
		case r >= 'a' && r <= 'z':
			hasLower = true
		case r >= '0' && r <= '9':
			hasNumber = true
		case r == '!' || r == '@' || r == '#' || r == '$' || r == '%' || r == '^' ||
			r == '&' || r == '*' || r == '(' || r == ')' || r == '-' || r == '_' ||
			r == '=' || r == '+' || r == '[' || r == ']' || r == '{' || r == '}' ||
			r == '|' || r == ';' || r == ':' || r == '"' || r == '\'' || r == ',' ||
			r == '.' || r == '<' || r == '>' || r == '/' || r == '?' || r == '~':
			hasSymbol = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSymbol {
		return fmt.Errorf("password must contain uppercase, lowercase, number, and symbol")
	}

	return nil
}