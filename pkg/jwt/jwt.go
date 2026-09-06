// Package jwt issues and parses the application JWT. Handlers and middleware
// depend on this package rather than github.com/golang-jwt/jwt/v5 directly
// so the signing scheme and claim shape live in one place.
package jwt

import (
	"time"

	gjwt "github.com/golang-jwt/jwt/v5"
)

// Claims is the single claim payload used for both hospital users and super
// admins. Super admin tokens set Super=true and leave TenantID empty.
type Claims struct {
	UserID   string `json:"uid"`
	TenantID string `json:"tid,omitempty"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Super    bool   `json:"super,omitempty"`
	gjwt.RegisteredClaims
}

// IssueUserToken mints a token for a hospital user.
func IssueUserToken(secret, userID, tenantID, email, role string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		RegisteredClaims: gjwt.RegisteredClaims{
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  gjwt.NewNumericDate(time.Now()),
			Issuer:    "hms",
		},
	}
	return gjwt.NewWithClaims(gjwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// IssueSuperToken mints a token for a platform super admin.
func IssueSuperToken(secret, adminID, email string, ttl time.Duration) (string, error) {
	claims := Claims{
		UserID: adminID,
		Email:  email,
		Role:   "super",
		Super:  true,
		RegisteredClaims: gjwt.RegisteredClaims{
			ExpiresAt: gjwt.NewNumericDate(time.Now().Add(ttl)),
			IssuedAt:  gjwt.NewNumericDate(time.Now()),
			Issuer:    "hms-super",
		},
	}
	return gjwt.NewWithClaims(gjwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// Parse verifies and returns the Claims from a raw token string.
func Parse(secret, raw string) (*Claims, error) {
	claims := &Claims{}
	tok, err := gjwt.ParseWithClaims(raw, claims, func(t *gjwt.Token) (any, error) {
		if _, ok := t.Method.(*gjwt.SigningMethodHMAC); !ok {
			return nil, gjwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !tok.Valid {
		return nil, gjwt.ErrSignatureInvalid
	}
	return claims, nil
}
