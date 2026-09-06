package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/super/model"
	"github.com/salman/hms-backend/internal/modules/super/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func buildMeta(c *fiber.Ctx) service.Meta {
	meta := service.Meta{IP: c.IP(), UserAgent: c.Get("User-Agent")}
	if claims, ok := auth.ClaimsFrom(c); ok && claims != nil {
		meta.ActorID = claims.UserID
		meta.ActorEmail = claims.Email
	}
	return meta
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	resp, err := h.svc.Login(c.Context(), buildMeta(c), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidBody):
			return response.BadRequest(c, "email and password required")
		case errors.Is(err, service.ErrInvalidCredentials):
			return response.Unauthorized(c, "invalid credentials")
		default:
			return response.Internal(c, "db error")
		}
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, resp)
}

func (h *Handler) ListTenants(c *fiber.Ctx) error {
	out, err := h.svc.ListTenants(c.Context())
	if err != nil {
		return response.Internal(c, "db error")
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"tenants": out})
}

func (h *Handler) CreateTenant(c *fiber.Ctx) error {
	var req model.CreateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	out, err := h.svc.CreateTenant(c.Context(), buildMeta(c), req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, out)
}

func (h *Handler) UpdateTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "tenant id required")
	}
	var req model.UpdateTenantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if err := h.svc.UpdateTenant(c.Context(), buildMeta(c), id, req); err != nil {
		if errors.Is(err, service.ErrNothingToUpdate) {
			return response.BadRequest(c, "nothing to update")
		}
		return response.Internal(c, err.Error())
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}

func (h *Handler) DeleteTenant(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return response.BadRequest(c, "tenant id required")
	}
	if err := h.svc.DeleteTenant(c.Context(), buildMeta(c), id); err != nil {
		return response.Internal(c, err.Error())
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}

func (h *Handler) SendSubscriptionReminder(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.SendSubscriptionReminder(c.Context(), buildMeta(c), id); err != nil {
		switch {
		case errors.Is(err, service.ErrTenantNotFound):
			return response.Error(c, fiber.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrNoAdminEmail):
			return response.BadRequest(c, err.Error())
		default:
			return response.Error(c, fiber.StatusBadGateway, err.Error())
		}
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}
