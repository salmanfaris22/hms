package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/upload/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Sign(c *fiber.Ctx) error {
	out, err := h.svc.Sign(c.Query("folder"))
	if err != nil {
		if errors.Is(err, service.ErrNotConfigured) {
			return response.Error(c, fiber.StatusServiceUnavailable, err.Error())
		}
		return response.Internal(c, "sign failed")
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, out)
}
