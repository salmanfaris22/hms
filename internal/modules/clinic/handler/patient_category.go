package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func (h *Handler) ListPatientCategories(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	cats, total, err := h.svc.ListPatientCategories(c.Context(), m, model.PCListFilter{
		ClinicID: c.Params("id"),
		Page:     page,
		Limit:    8,
		Search:   strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"categories": cats, "total": total, "page": page, "limit": 8,
	})
}

func (h *Handler) CreatePatientCategory(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreatePatientCategory(c.Context(), m, c.Params("id"), req.Name)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdatePatientCategory(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdatePatientCategory(c.Context(), m, c.Params("id"), c.Params("catId"), req.Name); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeletePatientCategory(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePatientCategory(c.Context(), m, c.Params("id"), c.Params("catId")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ListSpecialStatuses(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	statuses, total, err := h.svc.ListSpecialStatuses(c.Context(), m, model.PCListFilter{
		ClinicID: c.Params("id"),
		Page:     page,
		Limit:    8,
		Search:   strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"statuses": statuses, "total": total, "page": page, "limit": 8,
	})
}

func (h *Handler) CreateSpecialStatus(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.CreateSpecialStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateSpecialStatus(c.Context(), m, c.Params("id"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdateSpecialStatus(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.UpdateSpecialStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateSpecialStatus(c.Context(), m, c.Params("id"), c.Params("statusId"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteSpecialStatus(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteSpecialStatus(c.Context(), m, c.Params("id"), c.Params("statusId")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}
