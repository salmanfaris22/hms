package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/billing/model"
	"github.com/salman/hms-backend/internal/modules/billing/service"
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
		errors.Is(err, service.ErrNoItems),
		errors.Is(err, service.ErrBadStatus),
		errors.Is(err, service.ErrOverpay),
		errors.Is(err, service.ErrNoPatient),
		errors.Is(err, service.ErrSettled),
		errors.Is(err, service.ErrTooLarge),
		errors.Is(err, service.ErrNothingPaid),
		errors.Is(err, service.ErrRefundAmount),
		errors.Is(err, service.ErrNoAmount),
		errors.Is(err, service.ErrNotDraft):
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound),
		errors.Is(err, service.ErrNoDraft):
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

// ListInvoices — the Invoices tab (3564:55048).
func (h *Handler) ListInvoices(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.ListInvoices(c.Context(), meta,
		c.Query("clinic"), c.Query("q"), c.Query("status"),
		intQuery(c, "page", 1), intQuery(c, "pageSize", 10))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// Summary — the four cards above the table.
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

// ListPayments — the Payment History tab (3564:57666).
func (h *Handler) ListPayments(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.ListPayments(c.Context(), meta,
		c.Query("clinic"), c.Query("q"),
		intQuery(c, "page", 1), intQuery(c, "pageSize", 10))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// CreateInvoice — the New Invoice form (3564:58640) and its two siblings.
func (h *Handler) CreateInvoice(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.CreateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	out, err := h.svc.CreateInvoice(c.Context(), meta, req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// GetInvoice — one invoice, with its lines and payments.
func (h *Handler) GetInvoice(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.GetInvoice(c.Context(), meta, c.Query("clinic"), c.Params("id"))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// RecordPayment — taking money against an invoice already raised (3614:77115).
func (h *Handler) RecordPayment(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.PaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if req.ClinicID == "" {
		req.ClinicID = c.Query("clinic")
	}
	out, err := h.svc.RecordPayment(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// RecordRefund — Process Refund (4565:73203).
func (h *Handler) RecordRefund(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.RefundRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if req.ClinicID == "" {
		req.ClinicID = c.Query("clinic")
	}
	out, err := h.svc.RecordRefund(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// AppointmentDraft — the unfinished bill a visit already has, if any.
func (h *Handler) AppointmentDraft(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	out, err := h.svc.DraftForAppointment(c.Context(), meta, c.Query("clinic"), c.Query("appointment"))
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}

// UpdateInvoice — saving a draft that was opened again (4723:68314).
func (h *Handler) UpdateInvoice(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Error(c, fiber.StatusUnauthorized, "unauthorized")
	}
	var req model.CreateInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if req.ClinicID == "" {
		req.ClinicID = c.Query("clinic")
	}
	out, err := h.svc.UpdateInvoice(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return fail(c, err)
	}
	return response.OK(c, "OK", out)
}
