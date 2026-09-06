package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/patient/model"
	"github.com/salman/hms-backend/internal/modules/patient/service"
	"github.com/salman/hms-backend/pkg/constants"
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

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrTenantUnavail):
		return response.Internal(c, err.Error())
	case errors.Is(err, service.ErrForbidden):
		return response.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrNotFound):
		return response.NotFound(c, err.Error())
	case errors.Is(err, service.ErrDB):
		return response.Internal(c, err.Error())
	default:
		var ve service.ValidationError
		if errors.As(err, &ve) {
			return response.BadRequest(c, ve.Error())
		}
		return response.Internal(c, err.Error())
	}
}

func (h *Handler) List(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	clinicID := c.Query("clinic")
	if clinicID == "" {
		return response.BadRequest(c, "clinic is required")
	}
	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	res, err := h.svc.List(c.Context(), meta, model.ListFilter{
		ClinicID: clinicID,
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		Status:   strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, res)
}

func (h *Handler) Get(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	p, err := h.svc.Get(c.Context(), meta, c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, p)
}

func (h *Handler) Create(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, patientNumber, err := h.svc.Create(c.Context(), meta, req)
	if err != nil {
		if errors.Is(err, service.ErrForbidden) {
			return response.Forbidden(c, err.Error())
		}
		if errors.Is(err, service.ErrTenantUnavail) {
			return response.Internal(c, err.Error())
		}
		return response.BadRequest(c, err.Error())
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{
		"id":            id,
		"patientNumber": patientNumber,
	})
}

func (h *Handler) Update(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if err := h.svc.Update(c.Context(), meta, c.Params("id"), req); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "updated"})
}

func (h *Handler) Delete(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.Delete(c.Context(), meta, c.Params("id")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}

func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	docs, err := h.svc.ListDocuments(c.Context(), meta, c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"items": docs})
}

func (h *Handler) AddDocument(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.AddDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.AddDocument(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteDocument(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.DeleteDocument(c.Context(), meta, c.Params("id"), c.Params("docId")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}

func (h *Handler) AddMedication(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.AddMedicationRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.AddMedication(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) AddAlert(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.AddAlertRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.AddAlert(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteAlert(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.DeleteAlert(c.Context(), meta, c.Params("id")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}

func (h *Handler) AddAllergy(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.AddAllergyRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.AddAllergy(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteAllergy(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.DeleteAllergy(c.Context(), meta, c.Params("id")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}

func (h *Handler) SeedDemo(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.SeedDemo(c.Context(), meta, c.Params("id")); err != nil {
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "seeded"})
}

func (h *Handler) GetProfile(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	out, err := h.svc.GetProfile(c.Context(), meta, c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, out)
}

func (h *Handler) GetFieldConfig(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	clinicID := c.Query("clinic")
	if clinicID == "" {
		return response.BadRequest(c, "clinic is required")
	}
	cfg, err := h.svc.GetFieldConfig(c.Context(), meta, clinicID)
	if err != nil {
		return mapErr(c, err)
	}
	c.Set("Content-Type", "application/json")
	return c.Status(fiber.StatusOK).Send(cfg)
}

func (h *Handler) PutFieldConfig(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	clinicID := c.Query("clinic")
	if clinicID == "" {
		return response.BadRequest(c, "clinic is required")
	}
	var body json.RawMessage
	if err := c.BodyParser(&body); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	if err := h.svc.PutFieldConfig(c.Context(), meta, clinicID, body); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "ok"})
}

func (h *Handler) CollectPayment(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.CollectPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	out, err := h.svc.CollectPayment(c.Context(), meta, c.Params("id"), c.Params("invoiceId"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, out)
}

func (h *Handler) AddVitals(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.AddVitalsRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	n, err := h.svc.AddVitals(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"saved": n})
}

func (h *Handler) GetVitalsConfig(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	cfg, err := h.svc.VitalsConfig(c.Context(), meta, c.Query("clinic"))
	if err != nil {
		return mapErr(c, err)
	}
	c.Set("Content-Type", "application/json")
	return c.Send(cfg)
}

func (h *Handler) PutVitalsConfig(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.PutVitalsConfig(c.Context(), meta, c.Query("clinic"), c.Body()); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "saved"})
}

func (h *Handler) ListDocumentCategories(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	cats, err := h.svc.ListDocumentCategories(c.Context(), meta, c.Query("clinic"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"items": cats})
}

func (h *Handler) AddDocumentCategory(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.AddDocumentCategory(c.Context(), meta, c.Query("clinic"), req.Name)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteDocumentCategory(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.DeleteDocumentCategory(c.Context(), meta, c.Query("clinic"), c.Params("catId")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}

func (h *Handler) ListFamily(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	items, err := h.svc.ListFamily(c.Context(), meta, c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"items": items})
}

func (h *Handler) AddFamilyMember(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	var req model.AddFamilyLinkRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid body")
	}
	id, err := h.svc.AddFamilyMember(c.Context(), meta, c.Params("id"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusCreated, constants.MsgCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteFamilyMember(c *fiber.Ctx) error {
	meta, ok := buildMeta(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}
	if err := h.svc.DeleteFamilyMember(c.Context(), meta, c.Params("id"), c.Params("famId")); err != nil {
		return mapErr(c, err)
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, fiber.Map{"status": "deleted"})
}
