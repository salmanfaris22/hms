package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/logs/model"
)

type Repository struct {
	registry *pgxpool.Pool
}

func New(registry *pgxpool.Pool) *Repository { return &Repository{registry: registry} }

func (r *Repository) buildWhere(f model.ListFilter) (string, []any) {
	args := []any{f.TenantID}
	where := "tenant_id = $1::uuid"
	if f.User != "" {
		args = append(args, f.User)
		where += fmt.Sprintf(" AND actor_email = $%d", len(args))
	}
	if f.Category != "" {
		args = append(args, f.Category+".%")
		where += fmt.Sprintf(" AND action LIKE $%d", len(args))
	}
	if f.Search != "" {
		args = append(args, "%"+strings.ToLower(f.Search)+"%")
		i := len(args)
		where += fmt.Sprintf(
			" AND (lower(action) LIKE $%d OR lower(actor_email) LIKE $%d OR lower(resource) LIKE $%d)",
			i, i, i,
		)
	}
	if f.From != "" {
		args = append(args, f.From)
		where += fmt.Sprintf(" AND occurred_at >= $%d::date", len(args))
	}
	if f.To != "" {
		args = append(args, f.To)
		where += fmt.Sprintf(" AND occurred_at < ($%d::date + INTERVAL '1 day')", len(args))
	}
	return where, args
}

func (r *Repository) Count(ctx context.Context, f model.ListFilter) (int, error) {
	where, args := r.buildWhere(f)
	var total int
	err := r.registry.QueryRow(ctx,
		"SELECT count(*) FROM audit_events WHERE "+where, args...,
	).Scan(&total)
	return total, err
}

func (r *Repository) List(ctx context.Context, f model.ListFilter) ([]model.LogEntryDTO, error) {
	where, args := r.buildWhere(f)
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	rows, err := r.registry.Query(ctx,
		`SELECT id::text, occurred_at, actor_email, actor_kind, action, resource, ip, user_agent, metadata
		 FROM audit_events
		 WHERE `+where+`
		 ORDER BY occurred_at DESC
		 LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.LogEntryDTO{}
	for rows.Next() {
		var e model.LogEntryDTO
		var metaRaw []byte
		if err := rows.Scan(
			&e.ID, &e.OccurredAt, &e.ActorEmail, &e.ActorKind,
			&e.Action, &e.Resource, &e.IP, &e.UserAgent, &metaRaw,
		); err != nil {
			return nil, err
		}
		if len(metaRaw) > 0 {
			e.Metadata = json.RawMessage(metaRaw)
		} else {
			e.Metadata = json.RawMessage("{}")
		}
		out = append(out, e)
	}
	return out, nil
}

func (r *Repository) DistinctActors(ctx context.Context, tenantID string) ([]string, error) {
	rows, err := r.registry.Query(ctx,
		`SELECT DISTINCT actor_email FROM audit_events
		 WHERE tenant_id = $1::uuid AND actor_email != ''
		 ORDER BY actor_email`,
		tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []string{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err == nil && email != "" {
			users = append(users, email)
		}
	}
	return users, nil
}

func (r *Repository) ExportQuery(ctx context.Context, f model.ExportFilter) ([]model.ExportRow, error) {
	args := []any{f.TenantID}
	where := "tenant_id = $1::uuid"
	if f.From != "" {
		args = append(args, f.From)
		where += fmt.Sprintf(" AND occurred_at >= $%d::date", len(args))
	}
	if f.To != "" {
		args = append(args, f.To)
		where += fmt.Sprintf(" AND occurred_at < ($%d::date + INTERVAL '1 day')", len(args))
	}
	rows, err := r.registry.Query(ctx,
		`SELECT occurred_at, actor_email, actor_kind, action, resource, ip
		 FROM audit_events WHERE `+where+`
		 ORDER BY occurred_at DESC LIMIT 10000`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ExportRow{}
	for rows.Next() {
		var r model.ExportRow
		if err := rows.Scan(&r.OccurredAt, &r.ActorEmail, &r.ActorKind, &r.Action, &r.Resource, &r.IP); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
