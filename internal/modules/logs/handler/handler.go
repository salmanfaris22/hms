package handler

import (
	"encoding/csv"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/salman/hms-backend/internal/core/auth"
	"github.com/salman/hms-backend/internal/modules/logs/model"
	"github.com/salman/hms-backend/internal/modules/logs/service"
	"github.com/salman/hms-backend/pkg/constants"
	"github.com/salman/hms-backend/pkg/response"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) List(c *fiber.Ctx) error {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}

	page, _ := strconv.Atoi(c.Query("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	res, err := h.svc.List(c.Context(), model.ListFilter{
		TenantID: claims.TenantID,
		Page:     page,
		PageSize: pageSize,
		Search:   strings.TrimSpace(c.Query("search")),
		User:     strings.TrimSpace(c.Query("user")),
		Category: strings.TrimSpace(c.Query("category")),
		From:     strings.TrimSpace(c.Query("from")),
		To:       strings.TrimSpace(c.Query("to")),
	})
	if err != nil {
		log.Printf("logs list: %v", err)
		return response.Internal(c, "db error")
	}
	return response.Success(c, fiber.StatusOK, constants.MsgOK, res)
}

func (h *Handler) Export(c *fiber.Ctx) error {
	claims, ok := auth.ClaimsFrom(c)
	if !ok {
		return response.Unauthorized(c, "not authenticated")
	}

	rows, err := h.svc.Export(c.Context(), model.ExportFilter{
		TenantID: claims.TenantID,
		From:     strings.TrimSpace(c.Query("from")),
		To:       strings.TrimSpace(c.Query("to")),
	})
	if err != nil {
		return response.Internal(c, "db error")
	}

	c.Set("Content-Type", "text/csv; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="user_logs.csv"`)
	w := csv.NewWriter(c.Response().BodyWriter())
	_ = w.Write([]string{"Occurred At", "Actor Email", "Actor Kind", "Action", "Resource", "IP"})
	for _, r := range rows {
		_ = w.Write([]string{r.OccurredAt.Format(time.RFC3339), r.ActorEmail, r.ActorKind, r.Action, r.Resource, r.IP})
	}
	w.Flush()
	return nil
}
