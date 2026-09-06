package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/utils"
	appjwt "github.com/salman/hms-backend/pkg/jwt"
	appmw "github.com/salman/hms-backend/internal/core/middleware"
)

type Claims = appjwt.Claims

func HashPassword(plain string) (string, error) { return utils.HashPassword(plain) }
func CheckPassword(hash, plain string) bool     { return utils.CheckPassword(hash, plain) }

func IssueToken(secret, userID, tenantID, email, role string, ttl time.Duration) (string, error) {
	return appjwt.IssueUserToken(secret, userID, tenantID, email, role, ttl)
}
func IssueSuperToken(secret, adminID, email string, ttl time.Duration) (string, error) {
	return appjwt.IssueSuperToken(secret, adminID, email, ttl)
}

func Require(secret string) fiber.Handler { return appmw.Require(secret) }

func EnforceSuper(c *fiber.Ctx) error { return appmw.EnforceSuper(c) }

func ClaimsFrom(c *fiber.Ctx) (*Claims, bool) { return appmw.ClaimsFrom(c) }