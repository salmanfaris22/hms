package audit

import (
	"context"
	"encoding/json"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Kind string

const (
	ActorSuper  Kind = "super"
	ActorUser   Kind = "user"
	ActorSystem Kind = "system"
)

type Event struct {
	ActorID    string         // may be empty
	ActorEmail string
	ActorKind  Kind
	TenantID   string // may be empty
	Action     string // e.g. "tenant.create", "login.success"
	Resource   string
	Metadata   map[string]any
}

func Log(ctx context.Context, pool *pgxpool.Pool, c *fiber.Ctx, e Event) {
	var ip, ua string
	if c != nil {
		ip = c.IP()
		ua = c.Get("User-Agent")
	}
	LogRaw(ctx, pool, ip, ua, e)
}

// LogRaw records an audit event without a fiber.Ctx, for use in service layers.
func LogRaw(ctx context.Context, pool *pgxpool.Pool, ip, ua string, e Event) {
	meta := e.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	mdJSON, _ := json.Marshal(meta)
	kind := string(e.ActorKind)
	if kind == "" {
		kind = string(ActorUser)
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO audit_events (
			actor_id, actor_email, actor_kind, tenant_id,
			action, resource, ip, user_agent, metadata
		)
		VALUES (
			NULLIF($1,'')::uuid, $2, $3, NULLIF($4,'')::uuid,
			$5, $6, $7, $8, $9::jsonb
		)`,
		e.ActorID, e.ActorEmail, kind, e.TenantID,
		e.Action, e.Resource, ip, ua, string(mdJSON),
	)
	if err != nil {
		log.Printf("audit log failed: %v", err)
	}
}