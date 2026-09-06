package utils

import "fmt"

type ListQuery struct {
	BaseQuery    string
	Args         []any
	Page         int
	PageSize     int
	Total        int
	Stats        Stats
	FilteredCount int
}

type Stats struct {
	Total        int
	Active      int
	Inactive    int
	NewThisMonth int
}

// OptimizedPatientList combines stats + count + list into single query using CTE
const OptimizedPatientListQuery = `
	WITH stats AS (
		SELECT 
			count(*) FILTER (WHERE TRUE) as total,
			count(*) FILTER (WHERE status = 'active') as active,
			count(*) FILTER (WHERE status = 'inactive') as inactive,
			count(*) FILTER (WHERE created_at >= date_trunc('month', now())) as new_this_month
		FROM patients WHERE clinic_id = $1::uuid
	)
	SELECT 
		s.total, s.active, s.inactive, s.new_this_month,
		(SELECT count(*) FROM patients WHERE {{WHERE}}) as filtered_count,
		{{COLUMNS}}
	FROM stats s
	CROSS JOIN patients
	WHERE {{WHERE}}
	ORDER BY patients.created_at DESC
	LIMIT ${{LIMIT}} OFFSET ${{OFFSET}}
`

func BuildPatientListQuery(clinicID, status, search string, args []any, page, pageSize int) (string, []any) {
	args = append([]any{clinicID}, args...)
	where := "clinic_id = $1::uuid"
	argNum := 1

	if status == "active" || status == "inactive" {
		argNum++
		args = append(args, status)
		where += fmt.Sprintf(" AND status = $%d", argNum)
	}
	if search != "" {
		argNum++
		args = append(args, "%"+search+"%")
		where += fmt.Sprintf(" AND (lower(first_name) LIKE $%d OR lower(last_name) LIKE $%d OR lower(email) LIKE $%d OR lower(patient_number) LIKE $%d)", argNum, argNum, argNum, argNum)
	}

	limitNum := argNum + 1
	offsetNum := argNum + 2
	args = append(args, pageSize, (page-1)*pageSize)

	query := `
		WITH stats AS (
			SELECT 
				count(*) FILTER (WHERE TRUE) as total,
				count(*) FILTER (WHERE status = 'active') as active,
				count(*) FILTER (WHERE status = 'inactive') as inactive,
				count(*) FILTER (WHERE created_at >= date_trunc('month', now())) as new_this_month
			FROM patients WHERE clinic_id = $1::uuid
		)
		SELECT 
			s.total, s.active, s.inactive, s.new_this_month,
			(SELECT count(*) FROM patients WHERE ` + where + `) as filtered_count,
			p.id::text, p.clinic_id::text, p.patient_number,
			p.first_name, p.middle_name, p.last_name, p.date_of_birth, p.sex,
			p.blood_type, p.marital_status, p.photo_url,
			p.phones, p.email, p.address_line, p.city, p.state, p.country, p.zip_code,
			p.emergency_name, p.emergency_phone,
			p.allergies, p.medical_conditions,
			p.insurance_provider, p.insurance_id,
			p.condition, p.last_visit_at, p.status, p.notes, p.family_members,
			p.created_at, p.updated_at
		FROM stats s
		CROSS JOIN patients p
		WHERE ` + where + `
		ORDER BY p.created_at DESC
		LIMIT $` + Itoa(limitNum) + ` OFFSET $` + Itoa(offsetNum)

	return query, args
}