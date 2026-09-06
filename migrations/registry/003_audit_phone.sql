-- HIPAA-style audit trail + organisation contact phone.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS phone            TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone_country    TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS welcome_sent_at  TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS audit_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    actor_id    UUID,
    actor_email TEXT NOT NULL DEFAULT '',
    actor_kind  TEXT NOT NULL DEFAULT 'user', -- 'super' | 'user' | 'system'
    tenant_id   UUID,
    action      TEXT NOT NULL,                -- 'login.success', 'tenant.create', ...
    resource    TEXT NOT NULL DEFAULT '',
    ip          TEXT NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS audit_events_actor_idx  ON audit_events (actor_id);
CREATE INDEX IF NOT EXISTS audit_events_tenant_idx ON audit_events (tenant_id);
CREATE INDEX IF NOT EXISTS audit_events_action_idx ON audit_events (action);
CREATE INDEX IF NOT EXISTS audit_events_time_idx   ON audit_events (occurred_at DESC);
