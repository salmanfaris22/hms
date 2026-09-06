package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/appointment/model"
	"github.com/salman/hms-backend/internal/modules/appointment/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func buildMeta(c *fiber.Ctx) (service.Meta, bool) {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return service.Meta{}, false
	}
	return service.Meta{
		IP:         c.IP(),
		UserAgent:  c.Get("User-Agent"),
		ActorID:    claims.UserID,
		ActorEmail: claims.Email,
		TenantID:   claims.TenantID,
		UserID:     claims.UserID,
	}, true
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrTenantUnavail):
		return response.Internal(c, err.Error())
	case errors.Is(err, service.ErrForbidden):
		return response.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrBadTime), errors.Is(err, service.ErrNoUpdate):
		return response.BadRequest(c, err.Error())
	default:
		return response.Internal(c, err.Error())
	}
}

func (h *Handler) Book(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.BookRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.Book(c.Context(), meta, req)
	if err != nil {
		if errors.Is(err, service.ErrBadTime) ||
			err.Error() == "clinicId, patientId, scheduledAt are required" {
			return response.BadRequest(c, err.Error())
		}
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if err := h.svc.Update(c.Context(), meta, c.Params("id"), req); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}

func (h *Handler) GetSettings(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	clinicID := c.Query("clinic")
	if clinicID == "" {
		return response.BadRequest(c, "clinic is required")
	}
	out, err := h.svc.GetSettings(c.Context(), meta, clinicID)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, out)
}

func (h *Handler) PutSettings(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	clinicID := c.Query("clinic")
	if clinicID == "" {
		return response.BadRequest(c, "clinic is required")
	}
	var req model.PutSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if err := h.svc.PutSettings(c.Context(), meta, clinicID, req); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}
