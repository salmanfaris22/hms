package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func (h *Handler) ListTaxes(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	taxes, total, err := h.svc.ListTaxes(c.Context(), m, c.Params("id"), page)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"taxes": taxes, "total": total, "page": page, "limit": 8,
	})
}

func (h *Handler) CreateTax(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.CreateTaxRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateTax(c.Context(), m, c.Params("id"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdateTax(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.UpdateTaxRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateTax(c.Context(), m, c.Params("id"), c.Params("taxId"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) DeleteTax(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteTax(c.Context(), m, c.Params("id"), c.Params("taxId")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ListPaymentModes(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	modes, total, err := h.svc.ListPaymentModes(c.Context(), m, c.Params("id"), page)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"modes": modes, "total": total, "page": page, "limit": 8,
	})
}

func (h *Handler) CreatePaymentMode(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.CreatePaymentModeRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreatePaymentMode(c.Context(), m, c.Params("id"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) DeletePaymentMode(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePaymentMode(c.Context(), m, c.Params("id"), c.Params("modeId")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}
