package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (h *Handler) ListRadiologyTests(c *fiber.Ctx) error {
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
	out, total, err := h.svc.ListRadiologyTests(c.Context(), tenantID, userID, model.RadiologyListFilter{
		ClinicID: c.Query("clinic"),
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		Category: strings.TrimSpace(c.Query("category")),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"tests":    out,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *Handler) CreateRadiologyTest(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreateRadiologyTestRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateRadiologyTest(c.Context(), tenantID, userID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdateRadiologyTest(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreateRadiologyTestRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateRadiologyTest(c.Context(), tenantID, userID, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteRadiologyTest(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteRadiologyTest(c.Context(), tenantID, userID, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListRadiologyCategories(c *fiber.Ctx) error {
	return h.listLookup(c, "radiology_categories", "")
}
func (h *Handler) CreateRadiologyCategory(c *fiber.Ctx) error {
	return h.createLookup(c, "radiology_categories", "")
}
func (h *Handler) DeleteRadiologyCategory(c *fiber.Ctx) error {
	return h.deleteLookup(c, "radiology_categories")
}
