package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func (h *Handler) listOrgItems(c *fiber.Ctx, table string) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit := 8
	if l, err := strconv.Atoi(c.Query("limit", "")); err == nil && l > 0 && l <= 500 {
		limit = l
	}
	items, total, err := h.svc.ListOrgItems(c.Context(), m, table, model.OrgListFilter{
		ClinicID: c.Params("id"),
		Page:     page,
		Limit:    limit,
		Search:   strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"items": items, "total": total, "page": page, "limit": limit,
	})
}

func (h *Handler) createOrgItem(c *fiber.Ctx, table string) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.CreateOrgItemRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateOrgItem(c.Context(), m, table, c.Params("id"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) updateOrgItem(c *fiber.Ctx, table string) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.UpdateOrgItemRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateOrgItem(c.Context(), m, table, c.Params("id"), c.Params("itemId"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) deleteOrgItem(c *fiber.Ctx, table string) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteOrgItem(c.Context(), m, table, c.Params("id"), c.Params("itemId")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) CopyOrgSetup(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.CopyOrgRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	results, err := h.svc.CopyOrgSetup(c.Context(), m, c.Params("id"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"results": results})
}

func (h *Handler) ListDepartments(c *fiber.Ctx) error   { return h.listOrgItems(c, "departments") }
func (h *Handler) CreateDepartment(c *fiber.Ctx) error  { return h.createOrgItem(c, "departments") }
func (h *Handler) UpdateDepartment(c *fiber.Ctx) error  { return h.updateOrgItem(c, "departments") }
func (h *Handler) DeleteDepartment(c *fiber.Ctx) error  { return h.deleteOrgItem(c, "departments") }
func (h *Handler) ListSpecializations(c *fiber.Ctx) error  { return h.listOrgItems(c, "specializations") }
func (h *Handler) CreateSpecialization(c *fiber.Ctx) error { return h.createOrgItem(c, "specializations") }
func (h *Handler) UpdateSpecialization(c *fiber.Ctx) error { return h.updateOrgItem(c, "specializations") }
func (h *Handler) DeleteSpecialization(c *fiber.Ctx) error { return h.deleteOrgItem(c, "specializations") }
func (h *Handler) ListDesignations(c *fiber.Ctx) error  { return h.listOrgItems(c, "designations") }
func (h *Handler) CreateDesignation(c *fiber.Ctx) error { return h.createOrgItem(c, "designations") }
func (h *Handler) UpdateDesignation(c *fiber.Ctx) error { return h.updateOrgItem(c, "designations") }
func (h *Handler) DeleteDesignation(c *fiber.Ctx) error { return h.deleteOrgItem(c, "designations") }
