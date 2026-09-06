package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/dashboard/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Overview(c *fiber.Ctx) error {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "not authenticated")
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	period := c.Query("period", "week")

	out, err := h.svc.Overview(c.Context(), service.Meta{TenantID: claims.TenantID, UserID: claims.UserID}, clinicID, period)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			return response.Error(c, fiber.StatusForbidden, err.Error())
		case errors.Is(err, service.ErrTenantUnavail):
			return response.Error(c, fiber.StatusInternalServerError, "tenant unavailable")
		default:
			log.Printf("dashboard: %v", err)
			return response.Error(c, fiber.StatusInternalServerError, err.Error())
		}
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, out)
}
