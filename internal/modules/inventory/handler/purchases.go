package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (h *Handler) ListPurchases(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	purchases, total, err := h.svc.ListPurchases(c.Context(), tenantID, model.PurchaseListFilter{
		ClinicID: c.Query("clinicId"),
		Page:     page,
		Q:        strings.TrimSpace(c.Query("q")),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": purchases, "total": total, "page": page})
}

func (h *Handler) CreatePurchase(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreatePurchaseRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, purID, err := h.svc.CreatePurchase(c.Context(), tenantID, c.Query("clinicId"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id, "purchaseId": purID})
}

func (h *Handler) DeletePurchase(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeletePurchase(c.Context(), tenantID, c.Query("clinicId"), c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}
