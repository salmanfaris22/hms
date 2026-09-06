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
		// same shape as the patient module: a caller mistake is a 400, not a 500
		var ve service.ValidationError
		if errors.As(err, &ve) {
			return response.BadRequest(c, ve.Error())
		}
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

func (h *Handler) Calendar(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	day, err := h.svc.Calendar(c.Context(), meta, c.Query("clinic"), c.Query("from"), c.Query("to"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, day)
}

func (h *Handler) CreateBlock(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.CreateBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.CreateBlock(c.Context(), meta, c.Query("clinic"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteBlock(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.DeleteBlock(c.Context(), meta, c.Query("clinic"), c.Params("blockId")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.Delete(c.Context(), meta, c.Params("id")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}

func (h *Handler) PastVisits(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	out, err := h.svc.PastVisits(c.Context(), meta, c.Query("clinic"), c.Query("patient"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"visits": out})
}

func (h *Handler) GetPrescription(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	p, err := h.svc.Prescription(c.Context(), meta, c.Query("clinic"), c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, p)
}

func (h *Handler) PutPrescription(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.SavePrescriptionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.SavePrescription(c.Context(), meta, c.Query("clinic"), c.Params("id"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"id": id})
}
