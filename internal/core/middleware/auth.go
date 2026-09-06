package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/pkg/constants"
	appjwt "github.com/salman/hms-backend/pkg/jwt"
	"github.com/salman/hms-backend/pkg/response"
)

type ctxKey string

const claimsKey ctxKey = "jwt_claims"

func Require(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authz := c.Get("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			return response.Unauthorized(c, "Missing bearer token")
		}
		raw := strings.TrimPrefix(authz, "Bearer ")
		claims, err := appjwt.Parse(secret, raw)
		if err != nil {
			return response.Unauthorized(c, "Invalid token")
		}
		c.Locals(string(claimsKey), claims)
		return c.Next()
	}
}

func EnforceSuper(c *fiber.Ctx) error {
	claims, ok := c.Locals(string(claimsKey)).(*appjwt.Claims)
	if !ok || !claims.Super {
		return response.Forbidden(c, constants.MsgSuperAdminOnly)
	}
	return c.Next()
}

func ClaimsFrom(c *fiber.Ctx) (*appjwt.Claims, bool) {
	claims, ok := c.Locals(string(claimsKey)).(*appjwt.Claims)
	return claims, ok
}