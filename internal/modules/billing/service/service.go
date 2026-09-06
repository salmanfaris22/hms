package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/modules/billing/model"
	"github.com/salman/hms-backend/internal/modules/billing/repository"
	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
	ErrBadRequest    = errors.New("clinic is required")
	// things the caller got wrong, which are 400s rather than 500s
	ErrNoItems   = errors.New("an invoice needs at least one line")
	ErrBadStatus = errors.New("unknown status")
	ErrOverpay   = errors.New("payment is more than the invoice total")
	// the patient could not be identified — a caller error, not a server fault
	ErrNoPatient = errors.New("no patient matches that ID")
	// the two ways a payment against an existing invoice is refused
	ErrSettled  = errors.New("this invoice is already settled")
	ErrTooLarge = errors.New("payment is more than the balance due")
	// asked for an invoice this clinic does not have
	ErrNotFound = errors.New("invoice not found")
	// the two ways a refund is refused
	ErrNothingPaid  = errors.New("there is nothing paid on this invoice to refund")
	ErrRefundAmount = errors.New("refund is more than the net paid")
	ErrNoAmount     = errors.New("a refund needs an amount")
	// reopening or editing a draft bill
	ErrNoDraft  = errors.New("no draft bill for this appointment")
	ErrNotDraft = errors.New("only a draft can be edited")
)

type Meta struct {
	IP         string
	UserAgent  string
	ActorID    string
	ActorEmail string
	TenantID   string
	UserID     string
}

type Service struct {
	repo     *repository.Repository
	patients *patientService.Service
}

func New(repo *repository.Repository, patients *patientService.Service) *Service {
	return &Service{repo: repo, patients: patients}
}

// clamp keeps a page request inside something the database can answer quickly.
func clamp(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}

func (s *Service) ListInvoices(
	ctx context.Context, meta Meta, clinicID, q, status string, page, pageSize int,
) (model.InvoiceList, error) {
	if clinicID == "" {
		return model.InvoiceList{}, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return model.InvoiceList{}, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return model.InvoiceList{}, ErrForbidden
	}
	page, pageSize = clamp(page, pageSize)
	out, err := s.repo.ListInvoices(ctx, pool, clinicID, q, status, page, pageSize)
	if err != nil {
		log.Printf("list invoices: %v", err)
		return out, errors.New("db error")
	}
	// reading a clinic's billing is worth recording: it is money and it names patients
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.invoices.view", Resource: clinicID,
		Metadata: map[string]any{"count": len(out.Invoices)},
	})
	return out, nil
}

func (s *Service) Summary(ctx context.Context, meta Meta, clinicID string) (model.Summary, error) {
	if clinicID == "" {
		return model.Summary{}, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return model.Summary{}, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return model.Summary{}, ErrForbidden
	}
	out, err := s.repo.Summary(ctx, pool, clinicID)
	if err != nil {
		log.Printf("billing summary: %v", err)
		return out, errors.New("db error")
	}
	return out, nil
}

func (s *Service) ListPayments(
	ctx context.Context, meta Meta, clinicID, q string, page, pageSize int,
) (model.PaymentList, error) {
	if clinicID == "" {
		return model.PaymentList{}, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return model.PaymentList{}, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return model.PaymentList{}, ErrForbidden
	}
	page, pageSize = clamp(page, pageSize)
	out, err := s.repo.ListPayments(ctx, pool, clinicID, q, page, pageSize)
	if err != nil {
		log.Printf("list payments: %v", err)
		return out, errors.New("db error")
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.payments.view", Resource: clinicID,
		Metadata: map[string]any{"count": len(out.Payments)},
	})
	return out, nil
}

// ── Creating an invoice ─────────────────────────────────────────────────────

var validInvoiceStatus = map[string]bool{
	"pending": true, "paid": true, "draft": true, "proforma": true,
}

/*
 * Totals are computed here, never taken from the request.
 *
 * The browser shows the same arithmetic so the user is not surprised, but a
 * bill is money: the figure that gets stored has to be one the server worked
 * out itself from qty, rate and the two percentages. It mirrors
 * `frontend/src/features/invoice-create/totals.ts`, which is unit-tested.
 */
func clampPct(p float64) float64 {
	if p < 0 || p != p { // NaN never equals itself
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

type computed struct {
	taxable, tax, extraDiscount, extraTax, roundOff, grand int
	lines                                                  []map[string]any
}

func compute(req model.CreateInvoiceRequest) computed {
	var taxable, tax float64
	lines := make([]map[string]any, 0, len(req.Items))

	for _, it := range req.Items {
		qty := it.Qty
		if qty < 0 {
			qty = 0
		}
		rate := float64(it.RateCents)
		if rate < 0 {
			rate = 0
		}
		gross := qty * rate
		net := gross - gross*(clampPct(it.DiscountPct)/100)
		lineTax := net * (clampPct(it.TaxPct) / 100)
		taxable += net
		tax += lineTax
		lines = append(lines, map[string]any{
			"group": it.Group, "name": it.Name, "qty": qty,
			"rateCents": it.RateCents, "discountPct": it.DiscountPct, "taxPct": it.TaxPct,
			// what this line actually came to, so a stored invoice can be read
			// back without recomputing it
			"amount": int(net + lineTax + 0.5),
		})
	}

	discount := float64(req.ExtraDiscountCents)
	if discount < 0 {
		discount = 0
	}
	extraTax := float64(req.ExtraTaxCents)
	if extraTax < 0 {
		extraTax = 0
	}

	before := taxable + tax - discount + extraTax
	rounded := before
	switch req.RoundOff {
	case "nearest":
		rounded = float64(int(before/100+0.5)) * 100
	case "up":
		rounded = float64((int(before)+99)/100) * 100
	case "down":
		rounded = float64(int(before)/100) * 100
	}
	grand := int(rounded + 0.5)
	if grand < 0 {
		grand = 0
	}

	return computed{
		taxable: int(taxable + 0.5), tax: int(tax + 0.5),
		extraDiscount: int(discount), extraTax: int(extraTax),
		roundOff: int(rounded - before + 0.5), grand: grand, lines: lines,
	}
}

func (s *Service) CreateInvoice(
	ctx context.Context, meta Meta, req model.CreateInvoiceRequest,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult
	if req.ClinicID == "" {
		return out, ErrBadRequest
	}
	if len(req.Items) == 0 {
		return out, ErrNoItems
	}
	if req.Status == "" {
		req.Status = "pending"
	}
	if !validInvoiceStatus[req.Status] {
		return out, ErrBadStatus
	}

	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return out, ErrForbidden
	}

	c := compute(req)
	paid := 0
	for _, p := range req.Payments {
		if p.AmountCents > 0 {
			paid += p.AmountCents
		}
	}
	// a draft or a quotation is not money taken yet
	if req.Status == "draft" || req.Status == "proforma" {
		paid = 0
		req.Payments = nil
	}
	if paid > c.grand {
		return out, ErrOverpay
	}

	out, err = s.repo.CreateInvoice(ctx, pool, req, c.grand, paid, c.lines,
		c.extraDiscount, c.extraTax, c.roundOff)
	if err != nil {
		log.Printf("create invoice: %v", err)
		// say which part the caller got wrong rather than a blanket failure
		if errors.Is(err, repository.ErrPatientNotFound) {
			return out, fmt.Errorf("%w: %q", ErrNoPatient, req.PatientNumber)
		}
		return out, errors.New("could not save the invoice")
	}

	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.invoice.create", Resource: out.ID,
		Metadata: map[string]any{
			"number": out.Number, "totalCents": out.AmountTotalCents,
			"paidCents": out.AmountPaidCents, "status": out.Status, "lines": len(req.Items),
		},
	})
	return out, nil
}

// GetInvoice — one invoice opened from the list (3889:55439 and its siblings).
func (s *Service) GetInvoice(
	ctx context.Context, meta Meta, clinicID, id string,
) (model.InvoiceDetail, error) {
	var out model.InvoiceDetail
	if clinicID == "" || id == "" {
		return out, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return out, ErrForbidden
	}
	out, err = s.repo.GetInvoice(ctx, pool, clinicID, id)
	if err != nil {
		log.Printf("get invoice %s: %v", id, err)
		return out, ErrNotFound
	}
	// opening a bill shows what a patient was charged for, so it is recorded
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.invoice.view", Resource: id,
	})
	return out, nil
}

// RecordPayment — Make Payment on an invoice card (3889:55599 / 3823:69379).
func (s *Service) RecordPayment(
	ctx context.Context, meta Meta, invoiceID string, req model.PaymentRequest,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult
	if req.ClinicID == "" || invoiceID == "" {
		return out, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return out, ErrForbidden
	}

	out, err = s.repo.RecordPayment(ctx, pool, req.ClinicID, invoiceID, req)
	switch {
	case errors.Is(err, repository.ErrNothingDue):
		return out, ErrSettled
	case errors.Is(err, repository.ErrPaymentTooLarge):
		return out, ErrTooLarge
	case err != nil:
		log.Printf("record payment %s: %v", invoiceID, err)
		return out, errors.New("could not record the payment")
	}

	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.payment.record", Resource: out.Number,
	})
	return out, nil
}

// RecordRefund — Process Refund (4565:73203).
func (s *Service) RecordRefund(
	ctx context.Context, meta Meta, invoiceID string, req model.RefundRequest,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult
	if req.ClinicID == "" || invoiceID == "" {
		return out, ErrBadRequest
	}
	if req.AmountCents <= 0 {
		return out, ErrNoAmount
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return out, ErrForbidden
	}

	out, err = s.repo.RecordRefund(ctx, pool, invoiceID, req)
	switch {
	case errors.Is(err, repository.ErrNothingPaid):
		return out, ErrNothingPaid
	case errors.Is(err, repository.ErrRefundTooLarge):
		return out, ErrRefundAmount
	case err != nil:
		log.Printf("record refund %s: %v", invoiceID, err)
		return out, errors.New("could not record the refund")
	}

	// money leaving the till is the entry an auditor looks for first
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.refund.record", Resource: out.Number,
		Metadata: map[string]any{
			"amountCents": req.AmountCents, "method": req.Method, "note": req.Note,
		},
	})
	return out, nil
}

// DraftForAppointment — Generate Bill reopening the draft a visit already has.
func (s *Service) DraftForAppointment(
	ctx context.Context, meta Meta, clinicID, appointmentID string,
) (model.InvoiceDetail, error) {
	var out model.InvoiceDetail
	if clinicID == "" || appointmentID == "" {
		return out, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return out, ErrForbidden
	}
	id, err := s.repo.DraftForAppointment(ctx, pool, clinicID, appointmentID)
	if err != nil {
		return out, ErrNoDraft
	}
	out, err = s.repo.GetInvoice(ctx, pool, clinicID, id)
	if err != nil {
		log.Printf("draft for appointment %s: %v", appointmentID, err)
		return out, ErrNotFound
	}
	return out, nil
}

// UpdateInvoice — saving a draft that was opened again.
func (s *Service) UpdateInvoice(
	ctx context.Context, meta Meta, id string, req model.CreateInvoiceRequest,
) (model.CreateInvoiceResult, error) {
	var out model.CreateInvoiceResult
	if req.ClinicID == "" || id == "" {
		return out, ErrBadRequest
	}
	if len(req.Items) == 0 {
		return out, ErrNoItems
	}
	if req.Status == "" {
		req.Status = "draft"
	}
	if !validInvoiceStatus[req.Status] {
		return out, ErrBadStatus
	}

	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return out, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, req.ClinicID); err != nil {
		return out, ErrForbidden
	}

	c := compute(req)
	paid := 0
	for _, p := range req.Payments {
		if p.AmountCents > 0 {
			paid += p.AmountCents
		}
	}
	// a draft or a quotation is not money taken yet
	if req.Status == "draft" || req.Status == "proforma" {
		paid = 0
	}
	if paid > c.grand {
		return out, ErrOverpay
	}

	out, err = s.repo.UpdateInvoice(ctx, pool, id, req, c.grand, paid, c.lines,
		c.extraDiscount, c.extraTax, c.roundOff)
	switch {
	case errors.Is(err, repository.ErrNotDraft):
		return out, ErrNotDraft
	case err != nil:
		log.Printf("update invoice %s: %v", id, err)
		return out, errors.New("could not save the invoice")
	}

	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "billing.invoice.update", Resource: out.ID,
		Metadata: map[string]any{
			"number": out.Number, "totalCents": out.AmountTotalCents,
			"status": out.Status, "lines": len(req.Items),
		},
	})
	return out, nil
}
