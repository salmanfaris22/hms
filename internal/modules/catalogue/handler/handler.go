package handler

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/catalogue/model"
	"github.com/salman/hms-backend/internal/modules/catalogue/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) claims(c *fiber.Ctx) (string, string, error) {
	cl, ok := auth.ClaimsFrom(c)
	if !ok {
		return "", "", writeErr(c, fiber.StatusUnauthorized, "not authenticated")
	}
	return cl.TenantID, cl.UserID, nil
}

func mapSvcErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrTenantUnavail):
		return writeErr(c, fiber.StatusInternalServerError, "tenant unavailable")
	case errors.Is(err, service.ErrForbidden):
		return writeErr(c, fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrNotFound), errors.Is(err, service.ErrTestNotFound):
		return writeErr(c, fiber.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrClinicReq),
		errors.Is(err, service.ErrBadBody),
		errors.Is(err, service.ErrNameReq),
		errors.Is(err, service.ErrClinicNameReq),
		errors.Is(err, service.ErrClinicDrugReq),
		errors.Is(err, service.ErrDrugNameReq),
		errors.Is(err, service.ErrClinicTestReq),
		errors.Is(err, service.ErrTestNameReq),
		errors.Is(err, service.ErrClinicTreatReq),
		errors.Is(err, service.ErrTreatNameReq),
		errors.Is(err, service.ErrParamNameReq),
		errors.Is(err, service.ErrParamIDsReq),
		errors.Is(err, service.ErrGroupReq),
		errors.Is(err, service.ErrGroupNameReq),
		errors.Is(err, service.ErrCopyReq):
		return writeErr(c, fiber.StatusBadRequest, err.Error())
	default:
		log.Printf("catalogue: %v", err)
		return writeErr(c, fiber.StatusInternalServerError, err.Error())
	}
}

// ── drugs ────────────────────────────────────────────────────────────────────

func (h *Handler) ListDrugs(c *fiber.Ctx) error {
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
	out, total, err := h.svc.ListDrugs(c.Context(), tenantID, userID, model.DrugListFilter{
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
		"drugs":    out,
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *Handler) CreateDrug(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreateDrugRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateDrug(c.Context(), tenantID, userID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) UpdateDrug(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CreateDrugRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateDrug(c.Context(), tenantID, userID, c.Params("id"), req); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeleteDrug(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteDrug(c.Context(), tenantID, userID, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}

// ── lookups (categories, manufacturers, units) ────────────────────────────────

func (h *Handler) listLookup(c *fiber.Ctx, table, kind string) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	out, err := h.svc.ListLookup(c.Context(), tenantID, userID, table, c.Query("clinic"), kind)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"items": out})
}

func (h *Handler) createLookup(c *fiber.Ctx, table, kind string) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.LookupCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	id, name, err := h.svc.CreateLookup(c.Context(), tenantID, userID, table, kind, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, map[string]string{"id": id, "name": name})
}

func (h *Handler) deleteLookup(c *fiber.Ctx, table string) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteLookup(c.Context(), tenantID, userID, table, c.Params("id")); err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ListCategories(c *fiber.Ctx) error { return h.listLookup(c, "drug_categories", "") }
func (h *Handler) CreateCategory(c *fiber.Ctx) error { return h.createLookup(c, "drug_categories", "") }
func (h *Handler) DeleteCategory(c *fiber.Ctx) error { return h.deleteLookup(c, "drug_categories") }
func (h *Handler) ListManufacturers(c *fiber.Ctx) error {
	return h.listLookup(c, "drug_manufacturers", "")
}
func (h *Handler) CreateManufacturer(c *fiber.Ctx) error {
	return h.createLookup(c, "drug_manufacturers", "")
}
func (h *Handler) DeleteManufacturer(c *fiber.Ctx) error {
	return h.deleteLookup(c, "drug_manufacturers")
}
func (h *Handler) ListUnits(c *fiber.Ctx) error {
	kind := c.Query("kind")
	if kind != "primary" && kind != "secondary" {
		kind = "primary"
	}
	return h.listLookup(c, "drug_units", kind)
}
func (h *Handler) CreateUnit(c *fiber.Ctx) error {
	kind := c.Query("kind")
	if kind != "primary" && kind != "secondary" {
		kind = "primary"
	}
	return h.createLookup(c, "drug_units", kind)
}
func (h *Handler) DeleteUnit(c *fiber.Ctx) error { return h.deleteLookup(c, "drug_units") }

// ── prescription-pad terms ───────────────────────────────────────────────────

// The pad's sections each keep their own vocabulary; anything else is refused
// rather than silently filed under a kind nobody reads back.
var rxTermKinds = map[string]bool{
	"complaint":      true,
	"observation":    true,
	"diagnosis":      true,
	"treatment_plan": true,
	"treatment_done": true,
	"investigation":  true,
	"advice":         true,
}

func rxTermKind(c *fiber.Ctx) (string, bool) {
	kind := c.Query("kind")
	return kind, rxTermKinds[kind]
}

func (h *Handler) ListRxTerms(c *fiber.Ctx) error {
	kind, ok := rxTermKind(c)
	if !ok {
		return writeErr(c, fiber.StatusBadRequest, "unknown term kind")
	}
	return h.listLookup(c, "rx_terms", kind)
}

func (h *Handler) CreateRxTerm(c *fiber.Ctx) error {
	kind, ok := rxTermKind(c)
	if !ok {
		return writeErr(c, fiber.StatusBadRequest, "unknown term kind")
	}
	return h.createLookup(c, "rx_terms", kind)
}

func (h *Handler) DeleteRxTerm(c *fiber.Ctx) error { return h.deleteLookup(c, "rx_terms") }

// ── copy ─────────────────────────────────────────────────────────────────────

func (h *Handler) CopyCatalogue(c *fiber.Ctx) error {
	tenantID, userID, err := h.claims(c)
	if err != nil {
		return err
	}
	var req model.CopyCatalogueRequest
	if err := c.BodyParser(&req); err != nil {
		return writeErr(c, fiber.StatusBadRequest, "invalid body")
	}
	results, err := h.svc.CopyCatalogue(c.Context(), tenantID, userID, req)
	if err != nil {
		return mapSvcErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{"results": results})
}
