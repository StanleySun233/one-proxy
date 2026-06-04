package auth

import (
	"crypto/rand"
	"math/big"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash string, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func RandomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}
	lower := "abcdefghijklmnopqrstuvwxyz"
	upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits := "0123456789"
	special := "!@#$%^&*"
	all := lower + upper + digits + special

	password := make([]byte, 0, length)
	required := []string{lower, upper, digits, special}
	for _, charset := range required {
		char, err := randomChar(charset)
		if err != nil {
			return "", err
		}
		password = append(password, char)
	}
	for len(password) < length {
		char, err := randomChar(all)
		if err != nil {
			return "", err
		}
		password = append(password, char)
	}
	if err := shuffle(password); err != nil {
		return "", err
	}
	return string(password), nil
}

func randomChar(charset string) (byte, error) {
	max := big.NewInt(int64(len(charset)))
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return charset[value.Int64()], nil
}

func shuffle(value []byte) error {
	for i := len(value) - 1; i > 0; i-- {
		max := big.NewInt(int64(i + 1))
		pick, err := rand.Int(rand.Reader, max)
		if err != nil {
			return err
		}
		j := int(pick.Int64())
		value[i], value[j] = value[j], value[i]
	}
	return nil
}
