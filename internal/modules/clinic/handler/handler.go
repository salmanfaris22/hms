package handler

import (
	"errors"
	"log"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/clinic/model"
	"github.com/salman/hms-backend/internal/modules/clinic/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) meta(c *fiber.Ctx) (service.Meta, error) {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return service.Meta{}, writeErr(c, fiber.StatusUnauthorized, "not authenticated")
	}
	return service.Meta{
		IP:         c.IP(),
		UserAgent:  c.Get("User-Agent"),
		ActorID:    claims.UserID,
		ActorEmail: claims.Email,
		TenantID:   claims.TenantID,
		UserID:     claims.UserID,
	}, nil
}

func mapSvcErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrTenantUnavail):
		return writeErr(c, fiber.StatusInternalServerError, "tenant unavailable")
	case errors.Is(err, service.ErrForbidden),
		errors.Is(err, service.ErrSourceForbid):
		return writeErr(c, fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrNotFound),
		errors.Is(err, service.ErrGenericNotFound),
		errors.Is(err, service.ErrInviteNotFound):
		return writeErr(c, fiber.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrNameReq),
		errors.Is(err, service.ErrBadBody),
		errors.Is(err, service.ErrNothingUpdate),
		errors.Is(err, service.ErrBadInvite),
		errors.Is(err, service.ErrBadKind),
		errors.Is(err, service.ErrTargetsReq):
		return writeErr(c, fiber.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrBadTable):
		return writeErr(c, fiber.StatusInternalServerError, err.Error())
	default:
		log.Printf("clinic: %v", err)
		return writeErr(c, fiber.StatusInternalServerError, err.Error())
	}
}

func (h *Handler) ListAll(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	out, err := h.svc.ListAll(c.Context(), m, c.Query("archived") == "true")
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"clinics": out})
}

func (h *Handler) List(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	out, err := h.svc.List(c.Context(), m, c.Query("archived") == "true")
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"clinics": out})
}

func (h *Handler) Get(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinic, err := h.svc.Get(c.Context(), m, c.Params("id"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, clinic)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.CreateClinicRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.Create(c.Context(), m, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.UpdateClinicRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.Update(c.Context(), m, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Archive(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.Archive(c.Context(), m, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Restore(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.Restore(c.Context(), m, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Context(), m, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) MakePrimary(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.MakePrimary(c.Context(), m, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Join(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.JoinRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.Join(c.Context(), m, req.InviteCode)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"id": id})
}
