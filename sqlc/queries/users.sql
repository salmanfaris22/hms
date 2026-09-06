-- name: GetTenantBySlug :one
SELECT id, slug, name, created_at, updated_at
FROM tenants
WHERE slug = $1;

-- name: GetUserByEmail :one
SELECT id, tenant_id, email, password_hash, full_name, role, is_active, last_login_at, created_at, updated_at
FROM users
WHERE tenant_id = $1 AND email = $2 AND is_active = true;

-- name: FindUserAcrossTenants :one
SELECT u.id, u.tenant_id, u.email, u.password_hash, u.full_name, u.role, u.is_active
FROM users u
WHERE u.email = $1 AND u.is_active = true
LIMIT 1;

-- name: TouchUserLogin :exec
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = $1 AND tenant_id = $2;

-- name: CreateTenant :one
INSERT INTO tenants (slug, name)
VALUES ($1, $2)
RETURNING id, slug, name, created_at, updated_at;

-- name: CreateUser :one
INSERT INTO users (tenant_id, email, password_hash, full_name, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, tenant_id, email, full_name, role, is_active, created_at, updated_at;
