package service

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/modules/lab/model"
	"github.com/salman/hms-backend/internal/modules/lab/repository"
	patientService "github.com/salman/hms-backend/internal/modules/patient/service"
)

var (
	ErrTenantUnavail = errors.New("tenant unavailable")
	ErrForbidden     = errors.New("not a member of this clinic")
	ErrBadRequest    = errors.New("clinic is required")
	ErrNoPatient     = errors.New("an order needs a patient")
	ErrNoTests       = errors.New("an order needs at least one test")
	ErrBadPriority   = errors.New("unknown priority")
	ErrCollected     = errors.New("this sample has already been collected")
	ErrNotFound      = errors.New("test not found")
	ErrNoResult      = errors.New("a result needs a value or a flag")
	ErrNotCollected  = errors.New("collect the sample before entering a result")
)

// the three pills the form offers (3765:51732)
var validPriority = map[string]bool{"routine": true, "urgent": true, "stat": true}

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

func clamp(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return page, pageSize
}

// pool opens the tenant's database and confirms the caller belongs to the
// clinic they are asking about — every method starts here.
func (s *Service) pool(ctx context.Context, meta Meta, clinicID string) (*pgxpool.Pool, error) {
	if clinicID == "" {
		return nil, ErrBadRequest
	}
	pool, err := s.patients.Tenants().Pool(ctx, meta.TenantID)
	if err != nil {
		return nil, ErrTenantUnavail
	}
	if err := s.patients.AssertMember(ctx, pool, meta.UserID, clinicID); err != nil {
		return nil, ErrForbidden
	}
	return pool, nil
}

func (s *Service) ListTests(
	ctx context.Context, meta Meta, clinicID, q, category, status, priority, sample string,
	page, pageSize int,
) (model.OrderTestList, error) {
	var out model.OrderTestList
	pool, err := s.pool(ctx, meta, clinicID)
	if err != nil {
		return out, err
	}
	page, pageSize = clamp(page, pageSize)
	out, err = s.repo.ListTests(ctx, pool, clinicID, q, category, status, priority, sample, page, pageSize)
	if err != nil {
		log.Printf("list lab tests: %v", err)
		return out, errors.New("db error")
	}
	// a worklist names patients and what they are being tested for
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "lab.orders.view", Resource: clinicID,
	})
	return out, nil
}

func (s *Service) Summary(ctx context.Context, meta Meta, clinicID string) (model.Summary, error) {
	var out model.Summary
	pool, err := s.pool(ctx, meta, clinicID)
	if err != nil {
		return out, err
	}
	out, err = s.repo.Summary(ctx, pool, clinicID)
	if err != nil {
		log.Printf("lab summary: %v", err)
		return out, errors.New("db error")
	}
	return out, nil
}

func (s *Service) GetTest(
	ctx context.Context, meta Meta, clinicID, id string,
) (model.OrderTest, error) {
	var out model.OrderTest
	if id == "" {
		return out, ErrBadRequest
	}
	pool, err := s.pool(ctx, meta, clinicID)
	if err != nil {
		return out, err
	}
	out, err = s.repo.GetTest(ctx, pool, clinicID, id)
	if err != nil {
		log.Printf("get lab test %s: %v", id, err)
		return out, ErrNotFound
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "lab.order.view", Resource: id,
	})
	return out, nil
}

// CreateOrder — the New Test Order form (3765:51732).
func (s *Service) CreateOrder(
	ctx context.Context, meta Meta, req model.CreateOrderRequest,
) (model.CreateOrderResult, error) {
	var out model.CreateOrderResult
	if req.PatientID == "" {
		return out, ErrNoPatient
	}
	if len(req.Tests) == 0 {
		return out, ErrNoTests
	}
	if req.Priority == "" {
		req.Priority = "routine"
	}
	if !validPriority[req.Priority] {
		return out, ErrBadPriority
	}

	pool, err := s.pool(ctx, meta, req.ClinicID)
	if err != nil {
		return out, err
	}

	out, err = s.repo.CreateOrder(ctx, pool, req)
	switch {
	case errors.Is(err, repository.ErrNoTests):
		return out, ErrNoTests
	case err != nil:
		log.Printf("create lab order: %v", err)
		return out, errors.New("could not save the order")
	}

	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "lab.order.create", Resource: out.ID,
		Metadata: map[string]any{
			"number": out.Number, "tests": out.Tests,
			"priority": req.Priority, "patientId": req.PatientID,
		},
	})
	return out, nil
}

// Collect — Collect Sample (4565:70829).
func (s *Service) Collect(
	ctx context.Context, meta Meta, testID string, req model.CollectRequest,
) (model.OrderTest, error) {
	var out model.OrderTest
	if testID == "" {
		return out, ErrBadRequest
	}
	pool, err := s.pool(ctx, meta, req.ClinicID)
	if err != nil {
		return out, err
	}

	out, err = s.repo.Collect(ctx, pool, req.ClinicID, testID, req.CollectedBy)
	switch {
	case errors.Is(err, repository.ErrAlreadyCollected):
		return out, ErrCollected
	case err != nil:
		log.Printf("collect sample %s: %v", testID, err)
		return out, ErrNotFound
	}

	// who handled a specimen is a chain-of-custody question
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "lab.sample.collect", Resource: testID,
		Metadata: map[string]any{
			"collectedBy": req.CollectedBy, "test": out.TestName, "order": out.OrderNumber,
		},
	})
	return out, nil
}

// overallFlag decides what the report says as a whole.
//
// A panel is abnormal when any one of its parameters is; a test with no
// parameters carries whatever flag was set for it. The decision is made here
// rather than taken from the browser, because the Reports tab filters on it.
func overallFlag(req model.SaveResultRequest) string {
	for _, v := range req.Values {
		if v.Flag == "abnormal" {
			return "abnormal"
		}
	}
	if len(req.Values) == 0 && req.Flag == "abnormal" {
		return "abnormal"
	}
	return "normal"
}

// SaveResult — Enter Test Result (3911:56691 panel, 3911:56637 single).
func (s *Service) SaveResult(
	ctx context.Context, meta Meta, testID string, req model.SaveResultRequest,
) (model.OrderTest, error) {
	var out model.OrderTest
	if testID == "" {
		return out, ErrBadRequest
	}
	if len(req.Values) == 0 && req.Flag == "" {
		return out, ErrNoResult
	}
	pool, err := s.pool(ctx, meta, req.ClinicID)
	if err != nil {
		return out, err
	}

	flag := overallFlag(req)
	out, err = s.repo.SaveResult(ctx, pool, req.ClinicID, testID, req, flag)
	switch {
	case errors.Is(err, repository.ErrNotCollected):
		return out, ErrNotCollected
	case err != nil:
		log.Printf("save lab result %s: %v", testID, err)
		return out, ErrNotFound
	}

	// a result is a clinical finding about a person
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "lab.result.save", Resource: testID,
		Metadata: map[string]any{
			"report": out.ReportNumber, "flag": flag,
			"values": len(req.Values), "reportedBy": req.ReportedBy,
		},
	})
	return out, nil
}

// Values reads the parameters of a report back.
func (s *Service) Values(
	ctx context.Context, meta Meta, clinicID, testID string,
) ([]model.ResultValue, error) {
	if testID == "" {
		return nil, ErrBadRequest
	}
	pool, err := s.pool(ctx, meta, clinicID)
	if err != nil {
		return nil, err
	}
	out, err := s.repo.Values(ctx, pool, clinicID, testID)
	if err != nil {
		log.Printf("lab result values %s: %v", testID, err)
		return nil, ErrNotFound
	}
	return out, nil
}

// ListReports — the Reports tab (3765:52465).
func (s *Service) ListReports(
	ctx context.Context, meta Meta, clinicID, q, source, flag string, page, pageSize int,
) (model.ReportList, error) {
	var out model.ReportList
	pool, err := s.pool(ctx, meta, clinicID)
	if err != nil {
		return out, err
	}
	page, pageSize = clamp(page, pageSize)
	out, err = s.repo.ListReports(ctx, pool, clinicID, q, source, flag, page, pageSize)
	if err != nil {
		log.Printf("list lab reports: %v", err)
		return out, errors.New("db error")
	}
	audit.LogRaw(ctx, s.patients.Tenants().Registry(), meta.IP, meta.UserAgent, audit.Event{
		ActorID: meta.ActorID, ActorEmail: meta.ActorEmail, ActorKind: audit.ActorUser,
		TenantID: meta.TenantID, Action: "lab.reports.view", Resource: clinicID,
	})
	return out, nil
}
