package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	coreauth "github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/user/model"
	"github.com/salman/hms-backend/internal/modules/user/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if req.Email == "" || req.Password == "" {
		return response.BadRequest(c, "email and password required")
	}

	meta := model.RequestMeta{IP: c.IP(), UserAgent: c.Get("User-Agent")}
	resp, err := h.svc.Login(c.Context(), meta, req.Email, req.Password)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, resp)
}

func (h *Handler) Me(c *fiber.Ctx) error {
	claims, ok := c.Locals("jwt_claims").(*coreauth.Claims)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}

	dto, err := h.svc.Me(c.Context(), claims)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, dto)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		return response.Unauthorized(c, err.Error())
	case errors.Is(err, service.ErrOrgInactive), errors.Is(err, service.ErrSubscriptionExpired):
		return response.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrTenantUnavailable):
		return response.Internal(c, err.Error())
	default:
		log.Printf("user handler: %v", err)
		return response.Internal(c, "internal error")
	}
}
