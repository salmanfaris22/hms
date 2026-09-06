package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (h *Handler) DispenseSummary(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	s, err := h.svc.DispenseSummary(c.Context(), tenantID, c.Query("clinicId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, s)
}

func (h *Handler) ListDispenses(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	items, total, err := h.svc.ListDispenses(c.Context(), tenantID, model.DispenseListFilter{
		ClinicID:     c.Query("clinicId"),
		Page:         page,
		Q:            strings.TrimSpace(c.Query("q")),
		StatusFilter: c.Query("status"),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": items, "total": total, "page": page})
}

func (h *Handler) CreateDispense(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreateDispenseRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, dispNo, err := h.svc.CreateDispense(c.Context(), tenantID, c.Query("clinicId"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id, "dispenseNo": dispNo})
}

func (h *Handler) UpdateDispenseStatus(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateDispenseStatus(c.Context(), tenantID, c.Query("clinicId"), c.Params("id"), req.Status); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}
