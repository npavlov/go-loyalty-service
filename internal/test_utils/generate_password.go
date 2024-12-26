package testutils

import (
	"crypto/rand"
	"math/big"

	"github.com/pkg/errors"
)

func GeneratePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_+="
	if length <= 0 {
		return "", nil
	}

	password := make([]byte, length)
	for i := range password {
		char, err := randomChar(charset)
		if err != nil {
			return "", err
		}
		password[i] = char
	}

	return string(password), nil
}

func randomChar(charset string) (byte, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	if err != nil {
		return 0, errors.Wrap(err, "failed to generate random character")
	}

	return charset[n.Int64()], nil
}
