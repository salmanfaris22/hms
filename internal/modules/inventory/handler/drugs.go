package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/modules/inventory/model"
)

func (h *Handler) ListDrugs(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	drugs, total, err := h.svc.ListDrugs(c.Context(), tenantID, model.DrugListFilter{
		ClinicID:     c.Query("clinicId"),
		Page:         page,
		Q:            strings.TrimSpace(c.Query("q")),
		CategoryID:   c.Query("category"),
		StatusFilter: c.Query("status"),
		Sort:         c.Query("sort", "name"),
	})
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": drugs, "total": total, "page": page})
}

func (h *Handler) DrugSummary(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	s, err := h.svc.DrugSummary(c.Context(), tenantID, c.Query("clinicId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, s)
}

func (h *Handler) CreateDrug(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreateDrugRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateDrug(c.Context(), tenantID, c.Query("clinicId"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteDrug(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteDrug(c.Context(), tenantID, c.Query("clinicId"), c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}

func (h *Handler) ListCategories(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	cats, err := h.svc.ListCategories(c.Context(), tenantID, c.Query("clinicId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, cats)
}

func (h *Handler) CreateCategory(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "name required")
	}
	id, err := h.svc.CreateCategory(c.Context(), tenantID, c.Query("clinicId"), req.Name)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id, "name": req.Name})
}

func (h *Handler) DeleteCategory(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteCategory(c.Context(), tenantID, c.Query("clinicId"), c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}

func (h *Handler) ListManufacturers(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	mfrs, err := h.svc.ListManufacturers(c.Context(), tenantID, c.Query("clinicId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, mfrs)
}

func (h *Handler) CreateManufacturer(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "name required")
	}
	id, err := h.svc.CreateManufacturer(c.Context(), tenantID, c.Query("clinicId"), req.Name)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id, "name": req.Name})
}

func (h *Handler) DeleteManufacturer(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteManufacturer(c.Context(), tenantID, c.Query("clinicId"), c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, nil)
}

func (h *Handler) ListStock(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	batches, err := h.svc.ListStock(c.Context(), tenantID, c.Query("clinicId"), c.Query("drugId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, batches)
}

func (h *Handler) AddStock(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	var req model.CreateStockRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.AddStock(c.Context(), tenantID, c.Query("clinicId"), req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id})
}

func (h *Handler) SearchDrugsForPOS(c *fiber.Ctx) error {
	tenantID, err := h.tenantID(c)
	if err != nil {
		return err
	}
	items, err := h.svc.SearchPOS(c.Context(), tenantID, c.Query("clinicId"), strings.TrimSpace(c.Query("q")), c.Query("categoryId"))
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, items)
}
