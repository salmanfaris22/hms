package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/clinic/model"
)

func (h *Handler) getJSONSettings(c *fiber.Ctx, table string) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	raw, err := h.svc.GetJSONSettings(c.Context(), m, table, c.Params("id"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, json.RawMessage(raw))
}

func (h *Handler) putJSONSettings(c *fiber.Ctx, table string) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	body := c.Body()
	if err := h.svc.PutJSONSettings(c.Context(), m, table, c.Params("id"), body); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetCommunicationSettings(c *fiber.Ctx) error {
	return h.getJSONSettings(c, "communication_settings")
}
func (h *Handler) PutCommunicationSettings(c *fiber.Ctx) error {
	return h.putJSONSettings(c, "communication_settings")
}

func (h *Handler) GetIntegrationSettings(c *fiber.Ctx) error {
	return h.getJSONSettings(c, "integration_settings")
}
func (h *Handler) PutIntegrationSettings(c *fiber.Ctx) error {
	return h.putJSONSettings(c, "integration_settings")
}

func (h *Handler) GetPrintSettings(c *fiber.Ctx) error {
	return h.getJSONSettings(c, "print_settings")
}
func (h *Handler) PutPrintSettings(c *fiber.Ctx) error {
	return h.putJSONSettings(c, "print_settings")
}

func (h *Handler) GetNumberingSettings(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	out, err := h.svc.GetNumbering(c.Context(), m, c.Params("id"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, out)
}

func (h *Handler) PutNumberingSettings(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	var req model.NumberingSettings
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.PutNumbering(c.Context(), m, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "ok"})
}
