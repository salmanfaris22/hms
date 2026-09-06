package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/inventory/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// tenantID extracts tenant from JWT claims; writes 401 if missing.
func (h *Handler) tenantID(c *fiber.Ctx) (string, error) {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return "", writeErr(c, fiber.StatusUnauthorized, "not authenticated")
	}
	return claims.TenantID, nil
}

func mapSvcErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrTenantUnavail):
		return writeErr(c, fiber.StatusInternalServerError, "tenant unavailable")
	case errors.Is(err, service.ErrClinicReq):
		return writeErr(c, fiber.StatusBadRequest, "clinicId required")
	case errors.Is(err, service.ErrNameReq):
		return writeErr(c, fiber.StatusBadRequest, "name required")
	case errors.Is(err, service.ErrBadBody):
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	default:
		return writeErr(c, fiber.StatusInternalServerError, err.Error())
	}
}
