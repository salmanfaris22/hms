package handler

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/staff/model"
	"github.com/salman/hms-backend/internal/modules/staff/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) meta(c *fiber.Ctx) (service.Meta, error) {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return service.Meta{}, response.Error(c, fiber.StatusUnauthorized, "not authenticated")
	}
	return service.Meta{
		IP: c.IP(), UserAgent: c.Get("User-Agent"),
		ActorID: claims.UserID, ActorEmail: claims.Email,
		TenantID: claims.TenantID, UserID: claims.UserID,
	}, nil
}

func writeJSON(c *fiber.Ctx, status int, v any) error {
	msg := constants.MsgOK
	if status == fiber.StatusCreated {
		msg = constants.MsgCreated
	}
	return response.Success(c, status, msg, v)
}

func mapErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrTenantUnavail):
		return response.Error(c, fiber.StatusInternalServerError, "tenant unavailable")
	case errors.Is(err, service.ErrForbidden):
		return response.Error(c, fiber.StatusForbidden, err.Error())
	case errors.Is(err, service.ErrNotFound):
		return response.Error(c, fiber.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrBadBody),
		errors.Is(err, service.ErrNameReq),
		errors.Is(err, service.ErrEmailReq),
		errors.Is(err, service.ErrEmailUsed),
		errors.Is(err, service.ErrCodeUsed):
		return response.Error(c, fiber.StatusBadRequest, err.Error())
	default:
		log.Printf("staff: %v", err)
		return response.Error(c, fiber.StatusInternalServerError, err.Error())
	}
}

// ── Staff ───────────────────────────────────────────────────────────────────

func (h *Handler) ListStaff(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit > 200 {
		limit = 200
	}
	out, total, err := h.svc.ListStaff(c.Context(), m, clinicID, model.ListFilter{
		Page:         page,
		Limit:        limit,
		Search:       strings.TrimSpace(c.Query("q")),
		DepartmentID: strings.TrimSpace(c.Query("departmentId")),
		Status:       strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{
		"items": out, "total": total, "page": page, "limit": limit,
	})
}

func (h *Handler) GetStaff(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	dto, err := h.svc.GetStaff(c.Context(), m, clinicID, c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, dto)
}

func (h *Handler) CreateStaff(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	var req model.CreateStaffRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateStaff(c.Context(), m, clinicID, req)
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id})
}

func (h *Handler) UpdateStaff(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	var req model.UpdateStaffRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateStaff(c.Context(), m, clinicID, c.Params("id"), req); err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func (h *Handler) DeactivateStaff(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	if err := h.svc.DeactivateStaff(c.Context(), m, clinicID, c.Params("id")); err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func (h *Handler) Stats(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	out, err := h.svc.Stats(c.Context(), m, clinicID)
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, out)
}

// ── Roles ───────────────────────────────────────────────────────────────────

func (h *Handler) ListRoles(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	out, err := h.svc.ListRoles(c.Context(), m, clinicID)
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": out})
}

func (h *Handler) CreateRole(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	var req model.CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateRole(c.Context(), m, clinicID, req)
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id})
}

func (h *Handler) UpdateRole(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	var req model.UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.UpdateRole(c.Context(), m, clinicID, c.Params("id"), req); err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func (h *Handler) DeleteRole(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	if err := h.svc.DeleteRole(c.Context(), m, clinicID, c.Params("id")); err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

func (h *Handler) AssignRoles(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	var req model.AssignRolesRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	if err := h.svc.AssignRoles(c.Context(), m, clinicID, req); err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}

// ── Documents ───────────────────────────────────────────────────────────────

func (h *Handler) ListDocuments(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	out, err := h.svc.ListDocuments(c.Context(), m, clinicID, c.Params("id"))
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"items": out})
}

func (h *Handler) CreateDocument(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	var req model.CreateDocumentRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid body")
	}
	id, err := h.svc.CreateDocument(c.Context(), m, clinicID, c.Params("id"), req)
	if err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, fiber.Map{"id": id})
}

func (h *Handler) DeleteDocument(c *fiber.Ctx) error {
	m, err := h.meta(c)
	if err != nil {
		return err
	}
	clinicID := c.Query("clinicId")
	if clinicID == "" {
		return response.Error(c, fiber.StatusBadRequest, "clinicId is required")
	}
	if err := h.svc.DeleteDocument(c.Context(), m, clinicID, c.Params("id"), c.Params("docId")); err != nil {
		return mapErr(c, err)
	}
	return writeJSON(c, fiber.StatusOK, fiber.Map{"status": "ok"})
}
