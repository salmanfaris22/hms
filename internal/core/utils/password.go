// Package helper contains small pure utilities shared across layers —
// password hashing, random invite codes, fast integer formatting used by
// dynamic SQL builders.
package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash at the default cost.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword compares a bcrypt hash against a plain-text password.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
