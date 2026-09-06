package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (h *Handler) SalesSummary(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	s, err := h.svc.SalesSummary(c.Context(), tenantID, c.Query("clinicId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, s)
}

func (h *Handler) ListSales(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	sales, total, err := h.svc.ListSales(c.Context(), tenantID, model.SaleListFilter{
		ClinicID:     c.Query("clinicId"),
		Page:         page,
		Q:            strings.TrimSpace(c.Query("q")),
		StatusFilter: c.Query("status"),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": sales, "total": total, "page": page})
}

func (h *Handler) CreateSale(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreateSaleRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, invNo, err := h.svc.CreateSale(c.Context(), tenantID, c.Query("clinicId"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id, "invoiceNo": invNo})
}

func (h *Handler) DeleteSale(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteSale(c.Context(), tenantID, c.Query("clinicId"), c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}
