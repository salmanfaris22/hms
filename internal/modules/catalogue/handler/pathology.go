package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (h *Handler) ListPathologyTests(c *fiber.Ctx) error {
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
	out, total, err := h.svc.ListPathologyTests(c.Context(), tenantID, userID, model.PathologyListFilter{
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

func (h *Handler) CreatePathologyTest(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreatePathologyTestRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreatePathologyTest(c.Context(), tenantID, userID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdatePathologyTest(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreatePathologyTestRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdatePathologyTest(c.Context(), tenantID, userID, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) SyncPathologyTestParameters(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.SyncTestParamsRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	linked, err := h.svc.SyncTestParameters(c.Context(), tenantID, userID, c.Params("id"), req.ParamIDs)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]int{"linked": linked})
}

func (h *Handler) DeletePathologyTest(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePathologyTest(c.Context(), tenantID, userID, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListPathologyCategories(c *fiber.Ctx) error {
	return h.listLookup(c, "pathology_categories", "")
}

func (h *Handler) CreatePathologyCategory(c *fiber.Ctx) error {
	return h.createLookup(c, "pathology_categories", "")
}

func (h *Handler) DeletePathologyCategory(c *fiber.Ctx) error {
	return h.deleteLookup(c, "pathology_categories")
}

func (h *Handler) ListPathologyParameters(c *fiber.Ctx) error {
	tenantID, _, err := h.claims(c)
	if err != nil {
		return err
	}
	out, err := h.svc.ListPathologyParameters(c.Context(), tenantID, c.Query("test"), c.Query("clinic"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"items": out})
}

func (h *Handler) CreatePathologyParameter(c *fiber.Ctx) error {
	tenantID, _, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreatePathologyParamRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreatePathologyParameter(c.Context(), tenantID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdatePathologyParameter(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreatePathologyParamRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdatePathologyParameter(c.Context(), tenantID, userID, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeletePathologyParameter(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePathologyParameter(c.Context(), tenantID, userID, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListPathologyGroups(c *fiber.Ctx) error {
	tenantID, _, err := h.claims(c)
	if err != nil {
		return err
	}
	out, err := h.svc.ListPathologyGroups(c.Context(), tenantID, c.Query("clinic"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"items": out})
}

func (h *Handler) CreatePathologyGroup(c *fiber.Ctx) error {
	tenantID, _, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.SaveGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreatePathologyGroup(c.Context(), tenantID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdatePathologyGroup(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.SaveGroupRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdatePathologyGroup(c.Context(), tenantID, userID, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeletePathologyGroup(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePathologyGroup(c.Context(), tenantID, userID, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) RemovePathologyGroupParam(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.RemoveGroupParam(c.Context(), tenantID, userID, c.Params("id"), c.Params("paramId")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

// suppress unused import when no strconv usage remains
var _ = strconv.Itoa
