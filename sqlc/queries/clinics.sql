-- name: ListClinicsForUser :many
SELECT c.id, c.tenant_id, c.name, c.clinic_type, c.location, c.color,
       c.invite_code, c.created_at, c.updated_at,
       m.role, m.is_default, m.last_accessed_at
FROM clinics c
JOIN clinic_members m ON m.clinic_id = c.id
WHERE m.user_id = $1 AND c.tenant_id = $2
ORDER BY m.is_default DESC, c.created_at ASC;

-- name: CreateClinic :one
INSERT INTO clinics (tenant_id, name, clinic_type, location, color, invite_code)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, tenant_id, name, clinic_type, location, color, invite_code, created_at, updated_at;

-- name: AddClinicMember :exec
INSERT INTO clinic_members (clinic_id, user_id, role, is_default)
VALUES ($1, $2, $3, $4)
ON CONFLICT (clinic_id, user_id) DO NOTHING;

-- name: GetClinicByInviteCode :one
SELECT id, tenant_id, name, clinic_type, location, color, invite_code, created_at, updated_at
FROM clinics
WHERE invite_code = $1 AND tenant_id = $2;

-- name: TouchClinicAccess :exec
UPDATE clinic_members
SET last_accessed_at = now()
WHERE clinic_id = $1 AND user_id = $2;
