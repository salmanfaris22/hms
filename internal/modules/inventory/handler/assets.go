package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (h *Handler) AssetSummary(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	s, err := h.svc.AssetSummary(c.Context(), tenantID, c.Query("clinicId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, s)
}

func (h *Handler) ListAssets(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	assets, total, err := h.svc.ListAssets(c.Context(), tenantID, model.AssetListFilter{
		ClinicID:     c.Query("clinicId"),
		Page:         page,
		Q:            strings.TrimSpace(c.Query("q")),
		StatusFilter: c.Query("status"),
		CondFilter:   c.Query("condition"),
		CatFilter:    c.Query("category"),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": assets, "total": total, "page": page})
}

func (h *Handler) CreateAsset(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, assetNo, err := h.svc.CreateAsset(c.Context(), tenantID, c.Query("clinicId"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id, "assetNo": assetNo})
}

func (h *Handler) UpdateAsset(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateAsset(c.Context(), tenantID, c.Query("clinicId"), c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}

func (h *Handler) DeleteAsset(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteAsset(c.Context(), tenantID, c.Query("clinicId"), c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}
