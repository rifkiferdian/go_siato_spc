package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonMemory      = 64 * 1024
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLength  = 16
	argonKeyLength   = 32
)

var errInvalidPasswordHash = errors.New("format hash password Go tidak valid")

func HashPassword(password, pepper string) (string, error) {
	if len(pepper) < 32 {
		return "", errors.New("pepper password minimal 32 karakter")
	}
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("membuat salt password: %w", err)
	}
	key := derivePasswordKey(password, pepper, salt, argonMemory, argonIterations, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonIterations, argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func VerifyPassword(encodedHash, password, pepper string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, errInvalidPasswordHash
	}
	var memory uint32
	var iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false, errInvalidPasswordHash
	}
	if memory < 8*1024 || memory > 256*1024 || iterations < 1 || iterations > 10 || parallelism < 1 || parallelism > 16 {
		return false, errInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 || len(salt) > 64 {
		return false, errInvalidPasswordHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 || len(expected) > 64 {
		return false, errInvalidPasswordHash
	}
	actual := derivePasswordKey(password, pepper, salt, memory, iterations, parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}

func ValidateNewPassword(password string) error {
	if len(password) < 12 {
		return errors.New("kata sandi minimal 12 karakter")
	}
	var upper, lower, digit bool
	for _, character := range password {
		switch {
		case character >= 'A' && character <= 'Z':
			upper = true
		case character >= 'a' && character <= 'z':
			lower = true
		case character >= '0' && character <= '9':
			digit = true
		}
	}
	if !upper || !lower || !digit {
		return errors.New("kata sandi harus mengandung huruf besar, huruf kecil, dan angka")
	}
	return nil
}

func derivePasswordKey(password, pepper string, salt []byte, memory, iterations uint32, parallelism uint8, keyLength uint32) []byte {
	mac := hmac.New(sha256.New, []byte(pepper))
	_, _ = mac.Write([]byte(password))
	peppered := mac.Sum(nil)
	return argon2.IDKey(peppered, salt, iterations, memory, parallelism, keyLength)
}

func dummyPasswordWork(password, pepper string) {
	_ = derivePasswordKey(password, pepper, make([]byte, argonSaltLength), argonMemory, argonIterations, argonParallelism, argonKeyLength)
}
