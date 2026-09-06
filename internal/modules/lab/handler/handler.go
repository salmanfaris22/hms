package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/lab/model"
	"github.com/salman/hms-backend/internal/modules/lab/service"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func buildMeta(c *fiber.Ctx) (service.Meta, bool) {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return service.Meta{}, false
	}
	return service.Meta{
		IP:         c.IP(),
		UserAgent:  c.Get("User-Agent"),
		ActorID:    claims.UserID,
		ActorEmail: claims.Email,
		TenantID:   claims.TenantID,
		UserID:     claims.UserID,
	}, true
}

func fail(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrForbidden):
		return response.Error(c, fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrBadRequest),
		errors.Is(err, service.ErrNoPatient),
		errors.Is(err, service.ErrNoTests),
		errors.Is(err, service.ErrBadPriority),
		errors.Is(err, service.ErrCollected),
		errors.Is(err, service.ErrNoResult),
		errors.Is(err, service.ErrNotCollected):
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		return response.Error(c, fiber.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrTenantUnavail):
		return response.Error(c, fiber.StatusServiceUnavailable, err.Error())
	default:
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}
}

func intQuery(c *fiber.Ctx, key string, def int) int {
	if v, err := strconv.Atoi(c.Query(key)); err == nil {
		return v
	}
	return def
}

// ListTests — the Test Orders worklist (3765:51814).
func (h *Handler) ListTests(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.ListTests(c.Context(), meta,
		c.Query("clinic"), c.Query("q"), c.Query("category"), c.Query("status"),
		c.Query("priority"), c.Query("sample"),
		intQuery(c, "page", 1), intQuery(c, "pageSize", 10))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// Summary — the four cards above the worklist.
func (h *Handler) Summary(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.Summary(c.Context(), meta, c.Query("clinic"))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// CreateOrder — New Test Order (3765:51732).
func (h *Handler) CreateOrder(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if req.ClinicID == "" {
		req.ClinicID = c.Query("clinic")
	}
	out, err := h.svc.CreateOrder(c.Context(), meta, req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// GetTest — one row of the worklist.
func (h *Handler) GetTest(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.GetTest(c.Context(), meta, c.Query("clinic"), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// Collect — Collect Sample (4565:70829).
func (h *Handler) Collect(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.CollectRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if req.ClinicID == "" {
		req.ClinicID = c.Query("clinic")
	}
	out, err := h.svc.Collect(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// SaveResult — Enter Test Result (3911:56691 / 3911:56637).
func (h *Handler) SaveResult(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.SaveResultRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if req.ClinicID == "" {
		req.ClinicID = c.Query("clinic")
	}
	out, err := h.svc.SaveResult(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// Values — the parameters of one report.
func (h *Handler) Values(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.Values(c.Context(), meta, c.Query("clinic"), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", fiber.Map{"values": out})
}

// ListReports — the Reports tab (3765:52465).
func (h *Handler) ListReports(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.ListReports(c.Context(), meta,
		c.Query("clinic"), c.Query("q"), c.Query("source"), c.Query("flag"),
		intQuery(c, "page", 1), intQuery(c, "pageSize", 10))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}
