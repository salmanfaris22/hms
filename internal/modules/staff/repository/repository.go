package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
	"github.com/salman/hms-backend/internal/modules/staff/model"
)

type Repository struct {
	tenants *postgres.TenantResolver
}

func New(tenants *postgres.TenantResolver) *Repository {
	return &Repository{tenants: tenants}
}

func (r *Repository) Tenants() *postgres.TenantResolver { return r.tenants }
func (r *Repository) Pool(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	return r.tenants.Pool(ctx, tenantID)
}

func (r *Repository) IsMember(ctx context.Context, pool *pgxpool.Pool, clinicID, userID string) bool {
	var ok bool
	_ = pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM clinic_members WHERE clinic_id = $1::uuid AND user_id = $2::uuid)`,
		clinicID, userID,
	).Scan(&ok)
	return ok
}

// ── Staff list / count ─────────────────────────────────────────────────────

const staffColumns = `
    u.id::text,
    COALESCE(sp.staff_code, ''),
    u.email,
    u.full_name,
    COALESCE(sp.first_name, ''),
    COALESCE(sp.middle_name, ''),
    COALESCE(sp.last_name, ''),
    COALESCE(sp.profile_photo, ''),
    COALESCE(sp.department_id::text, ''),
    COALESCE(d.name, ''),
    COALESCE(sp.designation_id::text, ''),
    COALESCE(dg.name, ''),
    COALESCE(sp.specialization_id::text, ''),
    COALESCE(sx.name, ''),
    COALESCE(sp.mobile_country, u.phone_country),
    COALESCE(sp.mobile_number, u.phone),
    COALESCE(sp.additional_mobile, ''),
    COALESCE(sp.landline_number, ''),
    COALESCE(sp.view_in_emr, true),
    COALESCE(sp.join_date, u.created_at::date),
    COALESCE(sp.status, CASE WHEN u.is_active THEN 'active' ELSE 'inactive' END),
    u.is_active,
    u.is_super_admin,
    COALESCE(sp.father_name, ''),
    COALESCE(sp.mother_name, ''),
    COALESCE(sp.gender, ''),
    COALESCE(sp.marital_status, ''),
    sp.date_of_birth,
    COALESCE(sp.blood_group, ''),
    COALESCE(sp.professional_id, ''),
    COALESCE(sp.address, ''),
    COALESCE(sp.locality, ''),
    COALESCE(sp.pincode, ''),
    COALESCE(sp.state, ''),
    COALESCE(sp.country, ''),
    sp.date_of_leaving,
    COALESCE((SELECT string_agg(r.name, ', ' ORDER BY r.name)
              FROM user_roles ur JOIN roles r ON r.id = ur.role_id
              WHERE ur.user_id = u.id AND ur.clinic_id = cm.clinic_id), ''),
    -- an explicit stamp, or a login that has actually been used, both prove
    -- the account has working credentials
    (sp.credentials_set_at IS NOT NULL OR u.last_login_at IS NOT NULL)`

const staffFrom = `
FROM users u
JOIN clinic_members cm ON cm.user_id = u.id
LEFT JOIN staff_profiles sp ON sp.user_id = u.id
LEFT JOIN departments d   ON d.id = sp.department_id
LEFT JOIN designations dg ON dg.id = sp.designation_id
LEFT JOIN specializations sx ON sx.id = sp.specialization_id`

// clinicJoinScoped is the default: one row per member of the clinic being viewed.
const clinicJoinScoped = "JOIN clinic_members cm ON cm.user_id = u.id"

// clinicJoinAll is used when a super admin lists every user in the tenant. A plain
// LEFT JOIN would emit one row per membership, so someone in six clinics appeared
// six times and `total` counted them six times. DISTINCT ON collapses that to one
// row per user while keeping cm.clinic_id available for the roles sub-select.
const clinicJoinAll = `LEFT JOIN (
    SELECT DISTINCT ON (user_id) user_id, clinic_id
      FROM clinic_members
     ORDER BY user_id, clinic_id
) cm ON cm.user_id = u.id`

func clinicJoin(allUsers bool) string {
	if allUsers {
		return clinicJoinAll
	}
	return clinicJoinScoped
}

func staffSelect(join string) string {
	return "SELECT" + staffColumns + strings.Replace(staffFrom, clinicJoinScoped, join, 1)
}

func staffCountFrom(join string) string {
	return strings.Replace(staffFrom, clinicJoinScoped, join, 1)
}
func (r *Repository) ListStaff(ctx context.Context, pool *pgxpool.Pool, f model.ListFilter) ([]model.StaffDTO, int, error) {
	joinClinic := clinicJoin(f.AllUsers)
	where := []string{}
	args := []any{}
	if !f.AllUsers {
		where = append(where, "cm.clinic_id = $1::uuid")
		args = append(args, f.ClinicID)
	}
	if f.Search != "" {
		args = append(args, "%"+f.Search+"%")
		idx := strconv.Itoa(len(args))
		where = append(where, "(u.full_name ILIKE $"+idx+" OR u.email ILIKE $"+idx+" OR sp.staff_code ILIKE $"+idx+")")
	}
	if f.DepartmentID != "" {
		args = append(args, f.DepartmentID)
		where = append(where, "sp.department_id = $"+strconv.Itoa(len(args))+"::uuid")
	}
	if f.Status != "" {
		args = append(args, f.Status)
		where = append(where, "COALESCE(sp.status, CASE WHEN u.is_active THEN 'active' ELSE 'inactive' END) = $"+strconv.Itoa(len(args)))
	}

	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	from := staffCountFrom(joinClinic)

	var total int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) `+from+whereSQL, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := (f.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	q := staffSelect(joinClinic) + whereSQL +
		" ORDER BY u.created_at DESC LIMIT $" + strconv.Itoa(len(args)-1) +
		" OFFSET $" + strconv.Itoa(len(args))

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []model.StaffDTO{}
	for rows.Next() {
		var s model.StaffDTO
		if err := rows.Scan(
			&s.ID, &s.StaffCode, &s.Email, &s.FullName,
			&s.FirstName, &s.MiddleName, &s.LastName, &s.ProfilePhoto,
			&s.DepartmentID, &s.DepartmentName,
			&s.DesignationID, &s.DesignationName,
			&s.SpecializationID, &s.SpecializationName,
			&s.MobileCountry, &s.MobileNumber, &s.AdditionalMobile, &s.LandlineNumber,
			&s.ViewInEMR, &s.JoinDate, &s.Status, &s.IsActive, &s.IsSuperAdmin,
			&s.FatherName, &s.MotherName, &s.Gender, &s.MaritalStatus, &s.DateOfBirth,
			&s.BloodGroup, &s.ProfessionalID, &s.Address, &s.Locality, &s.Pincode,
			&s.State, &s.Country, &s.DateOfLeaving, &s.RoleNames, &s.HasCredentials,
		); err != nil {
			return nil, 0, err
		}
		s.WorkLocations = r.workLocations(ctx, pool, s.ID)
		out = append(out, s)
	}
	return out, total, nil
}

func (r *Repository) GetStaff(ctx context.Context, pool *pgxpool.Pool, clinicID, userID string) (*model.StaffDTO, error) {
	row := pool.QueryRow(ctx,
		staffSelect(clinicJoinScoped)+" WHERE u.id = $2::uuid AND cm.clinic_id = $1::uuid LIMIT 1",
		clinicID, userID,
	)
	var s model.StaffDTO
	if err := row.Scan(
		&s.ID, &s.StaffCode, &s.Email, &s.FullName,
		&s.FirstName, &s.MiddleName, &s.LastName, &s.ProfilePhoto,
		&s.DepartmentID, &s.DepartmentName,
		&s.DesignationID, &s.DesignationName,
		&s.SpecializationID, &s.SpecializationName,
		&s.MobileCountry, &s.MobileNumber, &s.AdditionalMobile, &s.LandlineNumber,
		&s.ViewInEMR, &s.JoinDate, &s.Status, &s.IsActive, &s.IsSuperAdmin,
		&s.FatherName, &s.MotherName, &s.Gender, &s.MaritalStatus, &s.DateOfBirth,
		&s.BloodGroup, &s.ProfessionalID, &s.Address, &s.Locality, &s.Pincode,
		&s.State, &s.Country, &s.DateOfLeaving, &s.RoleNames, &s.HasCredentials,
	); err != nil {
		return nil, err
	}
	s.WorkLocations = r.workLocations(ctx, pool, s.ID)
	return &s, nil
}

func (r *Repository) workLocations(ctx context.Context, pool *pgxpool.Pool, userID string) []string {
	rows, err := pool.Query(ctx,
		`SELECT clinic_id::text FROM staff_clinics WHERE user_id = $1::uuid`, userID,
	)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// ── Staff create / update / delete ─────────────────────────────────────────

func (r *Repository) NextStaffCode(ctx context.Context, pool *pgxpool.Pool) (string, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) + 1 FROM staff_profiles`,
	).Scan(&n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("S-%04d", n), nil
}

func (r *Repository) StaffCodeExists(ctx context.Context, pool *pgxpool.Pool, code string) bool {
	var exists bool
	_ = pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM staff_profiles WHERE staff_code = $1)`, code,
	).Scan(&exists)
	return exists
}

type CreateStaffParams struct {
	UserID           string
	Email            string
	PasswordHash     string
	FullName         string
	StaffCode        string
	FirstName        string
	MiddleName       string
	LastName         string
	ProfilePhoto     string
	DepartmentID     string
	DesignationID    string
	SpecializationID string
	MobileCountry    string
	MobileNumber     string
	AdditionalMobile string
	LandlineNumber   string
	ViewInEMR        bool
	WorkLocations    []string
	PrimaryClinicID  string
}

func nullable(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func (r *Repository) CreateStaff(ctx context.Context, pool *pgxpool.Pool, p CreateStaffParams) (string, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var userID string
	if p.UserID != "" {
		userID = p.UserID
		_, err = tx.Exec(ctx,
			`UPDATE users SET full_name = $2, phone = $3, phone_country = $4, updated_at = now() WHERE id = $1::uuid`,
			userID, p.FullName, p.MobileNumber, p.MobileCountry,
		)
		if err != nil {
			return "", err
		}
	} else {
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, full_name, role, phone, phone_country)
			VALUES ($1, $2, $3, 'staff', $4, $5)
			RETURNING id::text`,
			p.Email, p.PasswordHash, p.FullName, p.MobileNumber, p.MobileCountry,
		).Scan(&userID)
		if err != nil {
			return "", err
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO staff_profiles (
		    user_id, staff_code, profile_photo,
		    first_name, middle_name, last_name,
		    department_id, designation_id, specialization_id,
		    mobile_country, mobile_number, additional_mobile, landline_number,
		    view_in_emr
		) VALUES (
		    $1::uuid, $2, $3,
		    $4, $5, $6,
		    $7::uuid, $8::uuid, $9::uuid,
		    $10, $11, $12, $13,
		    $14
		)
		ON CONFLICT (user_id) DO UPDATE SET
		    staff_code = EXCLUDED.staff_code,
		    profile_photo = EXCLUDED.profile_photo,
		    first_name = EXCLUDED.first_name,
		    middle_name = EXCLUDED.middle_name,
		    last_name = EXCLUDED.last_name,
		    department_id = EXCLUDED.department_id,
		    designation_id = EXCLUDED.designation_id,
		    specialization_id = EXCLUDED.specialization_id,
		    mobile_country = EXCLUDED.mobile_country,
		    mobile_number = EXCLUDED.mobile_number,
		    additional_mobile = EXCLUDED.additional_mobile,
		    landline_number = EXCLUDED.landline_number,
		    view_in_emr = EXCLUDED.view_in_emr,
		    updated_at = now()`,
		userID, p.StaffCode, p.ProfilePhoto,
		p.FirstName, p.MiddleName, p.LastName,
		nullable(p.DepartmentID), nullable(p.DesignationID), nullable(p.SpecializationID),
		p.MobileCountry, p.MobileNumber, p.AdditionalMobile, p.LandlineNumber,
		p.ViewInEMR,
	)
	if err != nil {
		return "", err
	}

	allClinics := dedupe(append(append([]string{}, p.WorkLocations...), p.PrimaryClinicID))

	for _, cid := range allClinics {
		_, err = tx.Exec(ctx,
			`INSERT INTO clinic_members (clinic_id, user_id, role)
			 VALUES ($1::uuid, $2::uuid, 'staff')
			 ON CONFLICT (clinic_id, user_id) DO NOTHING`,
			cid, userID,
		)
		if err != nil {
			return "", err
		}
	}

	if _, err = tx.Exec(ctx, `DELETE FROM staff_clinics WHERE user_id = $1::uuid`, userID); err != nil {
		return "", err
	}
	for _, cid := range allClinics {
		_, err = tx.Exec(ctx,
			`INSERT INTO staff_clinics (user_id, clinic_id) VALUES ($1::uuid, $2::uuid)
			 ON CONFLICT DO NOTHING`,
			userID, cid,
		)
		if err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

func dedupe(in []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (r *Repository) UpdateStaff(ctx context.Context, pool *pgxpool.Pool, userID string, req model.UpdateStaffRequest) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	sets := []string{}
	args := []any{}
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
	}
	if req.FirstName != nil {
		add("first_name", *req.FirstName)
	}
	if req.MiddleName != nil {
		add("middle_name", *req.MiddleName)
	}
	if req.LastName != nil {
		add("last_name", *req.LastName)
	}
	if req.ProfilePhoto != nil {
		add("profile_photo", *req.ProfilePhoto)
	}
	if req.DepartmentID != nil {
		add("department_id", nullable(*req.DepartmentID))
	}
	if req.DesignationID != nil {
		add("designation_id", nullable(*req.DesignationID))
	}
	if req.SpecializationID != nil {
		add("specialization_id", nullable(*req.SpecializationID))
	}
	if req.MobileCountry != nil {
		add("mobile_country", *req.MobileCountry)
	}
	if req.MobileNumber != nil {
		add("mobile_number", *req.MobileNumber)
	}
	if req.AdditionalMobile != nil {
		add("additional_mobile", *req.AdditionalMobile)
	}
	if req.LandlineNumber != nil {
		add("landline_number", *req.LandlineNumber)
	}
	if req.ViewInEMR != nil {
		add("view_in_emr", *req.ViewInEMR)
	}
	if req.Status != nil {
		add("status", *req.Status)
	}

	if len(sets) > 0 {
		args = append(args, userID)
		q := "UPDATE staff_profiles SET " + strings.Join(sets, ", ") +
			", updated_at = now() WHERE user_id = $" + strconv.Itoa(len(args)) + "::uuid"
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return err
		}
	}

	// keep users.full_name in sync if name parts changed
	if req.FirstName != nil || req.MiddleName != nil || req.LastName != nil {
		if _, err := tx.Exec(ctx, `
			UPDATE users SET full_name = TRIM(BOTH ' ' FROM (
			    COALESCE((SELECT first_name FROM staff_profiles WHERE user_id = $1::uuid), '') || ' ' ||
			    COALESCE((SELECT middle_name FROM staff_profiles WHERE user_id = $1::uuid), '') || ' ' ||
			    COALESCE((SELECT last_name FROM staff_profiles WHERE user_id = $1::uuid), '')
			)), updated_at = now() WHERE id = $1::uuid`, userID,
		); err != nil {
			return err
		}
	}

	if req.WorkLocations != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM staff_clinics WHERE user_id = $1::uuid`, userID); err != nil {
			return err
		}
		for _, cid := range dedupe(*req.WorkLocations) {
			if _, err := tx.Exec(ctx,
				`INSERT INTO staff_clinics (user_id, clinic_id) VALUES ($1::uuid, $2::uuid) ON CONFLICT DO NOTHING`,
				userID, cid,
			); err != nil {
				return err
			}
		}

		if _, err := tx.Exec(ctx, `DELETE FROM clinic_members WHERE user_id = $1::uuid`, userID); err != nil {
			return err
		}
		for _, cid := range dedupe(*req.WorkLocations) {
			if _, err := tx.Exec(ctx,
				`INSERT INTO clinic_members (clinic_id, user_id, role)
				 VALUES ($1::uuid, $2::uuid, 'staff')
				 ON CONFLICT (clinic_id, user_id) DO NOTHING`,
				cid, userID,
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) DeactivateStaff(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`UPDATE users SET is_active = false, updated_at = now() WHERE id = $1::uuid`,
		userID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE staff_profiles SET status = 'inactive', updated_at = now() WHERE user_id = $1::uuid`,
		userID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ── Stats ───────────────────────────────────────────────────────────────────

func (r *Repository) Stats(ctx context.Context, pool *pgxpool.Pool, clinicID string) (model.StatsDTO, error) {
	var s model.StatsDTO
	err := pool.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM clinic_members cm JOIN users u ON u.id = cm.user_id WHERE cm.clinic_id = $1::uuid AND u.is_active = true),
		  (SELECT COUNT(*) FROM clinic_members cm JOIN users u ON u.id = cm.user_id WHERE cm.clinic_id = $1::uuid AND u.last_login_at::date = CURRENT_DATE),
		  (SELECT COUNT(*) FROM clinic_members cm JOIN staff_profiles sp ON sp.user_id = cm.user_id WHERE cm.clinic_id = $1::uuid AND sp.status = 'on_leave'),
		  (SELECT COUNT(*) FROM departments WHERE clinic_id = $1::uuid)`,
		clinicID,
	).Scan(&s.TotalStaff, &s.ActiveToday, &s.OnLeave, &s.Departments)
	return s, err
}

// ── Roles ───────────────────────────────────────────────────────────────────

func (r *Repository) ListRoles(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]model.RoleDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT r.id::text, r.name, r.description, r.permissions, r.created_at,
		    (SELECT COUNT(*) FROM user_roles ur WHERE ur.role_id = r.id AND ur.clinic_id = r.clinic_id)
		FROM roles r WHERE r.clinic_id = $1::uuid
		ORDER BY r.created_at ASC`, clinicID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.RoleDTO{}
	for rows.Next() {
		var r model.RoleDTO
		var permsRaw []byte
		var createdAt time.Time
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &permsRaw, &createdAt, &r.StaffCount); err != nil {
			return nil, err
		}
		r.CreatedAt = createdAt
		_ = json.Unmarshal(permsRaw, &r.Permissions)
		if r.Permissions == nil {
			r.Permissions = []string{}
		}
		out = append(out, r)
	}
	return out, nil
}

func (r *Repository) CreateRole(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreateRoleRequest) (string, error) {
	perms, _ := json.Marshal(req.Permissions)
	if len(perms) == 0 {
		perms = []byte(`[]`)
	}
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO roles (clinic_id, name, description, permissions)
		VALUES ($1::uuid, $2, $3, $4::jsonb)
		RETURNING id::text`,
		clinicID, req.Name, req.Description, perms,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdateRole(ctx context.Context, pool *pgxpool.Pool, clinicID, roleID string, req model.UpdateRoleRequest) error {
	sets := []string{}
	args := []any{}
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, col+" = $"+strconv.Itoa(len(args)))
	}
	if req.Name != nil {
		add("name", *req.Name)
	}
	if req.Description != nil {
		add("description", *req.Description)
	}
	if req.Permissions != nil {
		b, _ := json.Marshal(*req.Permissions)
		args = append(args, string(b))
		sets = append(sets, "permissions = $"+strconv.Itoa(len(args))+"::jsonb")
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, roleID, clinicID)
	q := "UPDATE roles SET " + strings.Join(sets, ", ") +
		", updated_at = now() WHERE id = $" + strconv.Itoa(len(args)-1) + "::uuid" +
		" AND clinic_id = $" + strconv.Itoa(len(args)) + "::uuid"
	_, err := pool.Exec(ctx, q, args...)
	return err
}

func (r *Repository) DeleteRole(ctx context.Context, pool *pgxpool.Pool, clinicID, roleID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM roles WHERE id = $1::uuid AND clinic_id = $2::uuid`,
		roleID, clinicID,
	)
	return err
}

func (r *Repository) AssignRoles(ctx context.Context, pool *pgxpool.Pool, clinicID string, userIDs, roleIDs []string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, uid := range userIDs {
		if _, err := tx.Exec(ctx,
			`DELETE FROM user_roles WHERE user_id = $1::uuid AND clinic_id = $2::uuid`,
			uid, clinicID,
		); err != nil {
			return err
		}
		for _, rid := range roleIDs {
			if _, err := tx.Exec(ctx,
				`INSERT INTO user_roles (user_id, role_id, clinic_id)
				 VALUES ($1::uuid, $2::uuid, $3::uuid)
				 ON CONFLICT DO NOTHING`,
				uid, rid, clinicID,
			); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

// ── Documents ───────────────────────────────────────────────────────────────

func (r *Repository) ListDocuments(ctx context.Context, pool *pgxpool.Pool, userID string) ([]model.DocumentDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, name, url, mime_type, size_bytes, created_at
		FROM staff_documents WHERE user_id = $1::uuid ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.DocumentDTO{}
	for rows.Next() {
		var d model.DocumentDTO
		if err := rows.Scan(&d.ID, &d.Name, &d.URL, &d.MimeType, &d.SizeBytes, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *Repository) CreateDocument(ctx context.Context, pool *pgxpool.Pool, userID string, req model.CreateDocumentRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO staff_documents (user_id, name, url, mime_type, size_bytes)
		VALUES ($1::uuid, $2, $3, $4, $5)
		RETURNING id::text`,
		userID, req.Name, req.URL, req.MimeType, req.SizeBytes,
	).Scan(&id)
	return id, err
}

func (r *Repository) DeleteDocument(ctx context.Context, pool *pgxpool.Pool, userID, docID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM staff_documents WHERE id = $1::uuid AND user_id = $2::uuid`,
		docID, userID,
	)
	return err
}

// ── Directory (registry) ────────────────────────────────────────────────────

func (r *Repository) UpsertDirectory(ctx context.Context, email, tenantID, userID string) error {
	_, err := r.tenants.Registry().Exec(ctx, `
		INSERT INTO user_directory (email, tenant_id, user_id)
		VALUES ($1, $2::uuid, $3::uuid)
		ON CONFLICT (email) DO UPDATE
		SET tenant_id = EXCLUDED.tenant_id, user_id = EXCLUDED.user_id`,
		email, tenantID, userID,
	)
	return err
}

func (r *Repository) DirectoryLookup(ctx context.Context, email string) (string, string, error) {
	var tenantID, userID string
	err := r.tenants.Registry().QueryRow(ctx,
		`SELECT tenant_id::text, user_id::text FROM user_directory WHERE email = $1`,
		email,
	).Scan(&tenantID, &userID)
	return tenantID, userID, err
}

// PasswordHash returns the stored bcrypt hash for a user, so a password change
// can be checked against the current one.
func (r *Repository) PasswordHash(ctx context.Context, pool *pgxpool.Pool, userID string) (string, error) {
	var hash string
	err := pool.QueryRow(ctx,
		`SELECT COALESCE(password_hash, '') FROM users WHERE id = $1::uuid`, userID,
	).Scan(&hash)
	return hash, err
}

func (r *Repository) SetCredentials(ctx context.Context, pool *pgxpool.Pool, userID, email, passwordHash string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE users SET email = $2, password_hash = $3, is_active = true, updated_at = now()
		WHERE id = $1::uuid`,
		userID, email, passwordHash,
	); err != nil {
		return err
	}
	// marks the account as having a chosen login rather than the default issued
	// at creation, which is what HasCredentials reports
	if _, err := tx.Exec(ctx, `
		UPDATE staff_profiles SET credentials_set_at = now(), updated_at = now()
		WHERE user_id = $1::uuid`,
		userID,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) SyncClinicMembership(ctx context.Context, pool *pgxpool.Pool, userID string, clinicIDs []string) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM clinic_members WHERE user_id = $1::uuid`, userID); err != nil {
		return err
	}
	for _, cid := range clinicIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO clinic_members (clinic_id, user_id, role)
			VALUES ($1::uuid, $2::uuid, 'staff')
			ON CONFLICT (clinic_id, user_id) DO NOTHING`,
			cid, userID,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListSchedule(ctx context.Context, pool *pgxpool.Pool, userID string) ([]model.ScheduleSlot, error) {
	rows, err := pool.Query(ctx, `
		SELECT id::text, day_of_week, start_time::text, end_time::text, is_active
		FROM staff_weekly_schedule WHERE user_id = $1::uuid
		ORDER BY day_of_week, start_time`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ScheduleSlot{}
	for rows.Next() {
		var s model.ScheduleSlot
		if err := rows.Scan(&s.ID, &s.DayOfWeek, &s.StartTime, &s.EndTime, &s.IsActive); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *Repository) UpdateSchedule(ctx context.Context, pool *pgxpool.Pool, userID string, slots []model.ScheduleSlot) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM staff_weekly_schedule WHERE user_id = $1::uuid`, userID); err != nil {
		return err
	}
	for _, s := range slots {
		if _, err := tx.Exec(ctx, `
			INSERT INTO staff_weekly_schedule (user_id, day_of_week, start_time, end_time, is_active)
			VALUES ($1::uuid, $2, $3::time, $4::time, $5)`,
			userID, s.DayOfWeek, s.StartTime, s.EndTime, s.IsActive,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) GetFieldConfig(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]byte, error) {
	var cfg []byte
	err := pool.QueryRow(ctx,
		`SELECT config FROM staff_field_config WHERE clinic_id = $1::uuid`,
		clinicID,
	).Scan(&cfg)
	if err == pgx.ErrNoRows {
		return []byte("{}"), nil
	}
	return cfg, err
}

func (r *Repository) PutFieldConfig(ctx context.Context, pool *pgxpool.Pool, clinicID string, config json.RawMessage) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO staff_field_config (clinic_id, config, updated_at)
		VALUES ($1::uuid, $2::jsonb, now())
		ON CONFLICT (clinic_id) DO UPDATE SET config = EXCLUDED.config, updated_at = now()`,
		clinicID, config,
	)
	return err
}
