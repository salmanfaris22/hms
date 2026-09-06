package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (h *Handler) ListTreatments(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	out, total, err := h.svc.ListTreatments(c.Context(), tenantID, userID, model.TreatmentListFilter{
		ClinicID: c.Query("clinic"),
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		Sort:     strings.ToLower(strings.TrimSpace(c.Query("sort"))),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"treatments": out,
		"page":       page,
		"pageSize":   pageSize,
		"total":      total,
	})
}

func (h *Handler) CreateTreatment(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreateTreatmentRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateTreatment(c.Context(), tenantID, userID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdateTreatment(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreateTreatmentRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateTreatment(c.Context(), tenantID, userID, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteTreatment(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteTreatment(c.Context(), tenantID, userID, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}
