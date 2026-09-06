package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (r *Repository) ListPathologyTests(ctx context.Context, pool *pgxpool.Pool, f model.PathologyListFilter) ([]model.PathologyTestDTO, int, error) {
	args := []any{f.ClinicID}
	where := "pt.clinic_id = $1::uuid"
	argIdx := 2
	if f.Category != "" {
		args = append(args, f.Category)
		where += " AND category = $" + strconv.Itoa(argIdx)
		argIdx++
	}
	if f.Search != "" {
		searchLower := strings.ToLower(f.Search)
		args = append(args, searchLower, searchLower, searchLower)
		where += fmt.Sprintf(" AND (lower(test_name) LIKE $%d OR lower(test_code) LIKE $%d OR lower(category) LIKE $%d)", argIdx, argIdx+1, argIdx+2)
		argIdx += 3
	}

	var total int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM pathology_tests pt WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	query := `
		SELECT pt.id::text, pt.clinic_id::text, pt.test_name, pt.test_code, pt.category,
		       COALESCE(pt.test_type, '') as sample_type, pt.price, pt.tax,
		       (SELECT count(*) FROM pathology_test_parameters ptp WHERE ptp.test_id = pt.id) as param_count,
		       pt.created_at::text, pt.updated_at::text
		FROM pathology_tests pt
		WHERE ` + where + `
		ORDER BY pt.created_at DESC
		LIMIT $` + strconv.Itoa(limitIdx) + ` OFFSET $` + strconv.Itoa(offsetIdx)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []model.PathologyTestDTO{}
	for rows.Next() {
		var t model.PathologyTestDTO
		if err := rows.Scan(
			&t.ID, &t.ClinicID, &t.TestName, &t.TestCode, &t.Category,
			&t.SampleType, &t.Price, &t.Tax, &t.ParameterCount,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, nil
}

func (r *Repository) CreatePathologyTest(ctx context.Context, pool *pgxpool.Pool, req model.CreatePathologyTestRequest) (string, error) {
	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO pathology_tests (clinic_id, test_name, test_code, category, test_type, price, tax)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		RETURNING id::text`,
		req.ClinicID, req.TestName, req.TestCode, req.Category, req.SampleType, req.Price, req.Tax,
	).Scan(&id)
	return id, err
}

func (r *Repository) UpdatePathologyTest(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.CreatePathologyTestRequest) (int64, error) {
	ct, err := pool.Exec(ctx, `
		UPDATE pathology_tests SET
			test_name = $3, test_code = $4, category = $5, test_type = $6,
			price = $7, tax = $8, updated_at = now()
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
		req.TestName, req.TestCode, req.Category, req.SampleType, req.Price, req.Tax,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) TestBelongsToUserClinic(ctx context.Context, pool *pgxpool.Pool, testID, userID string) (bool, error) {
	var ok bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM pathology_tests pt
		JOIN clinic_members cm ON cm.clinic_id = pt.clinic_id
		WHERE pt.id = $1::uuid AND cm.user_id = $2::uuid
	)`, testID, userID).Scan(&ok)
	return ok, err
}

func (r *Repository) SyncTestParameters(ctx context.Context, pool *pgxpool.Pool, testID string, paramIDs []string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`DELETE FROM pathology_test_parameters WHERE test_id = $1::uuid`, testID,
	); err != nil {
		return err
	}
	for _, pid := range paramIDs {
		if pid == "" {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO pathology_test_parameters (test_id, param_id)
			 VALUES ($1::uuid, $2::uuid)
			 ON CONFLICT (test_id, param_id) DO NOTHING`,
			testID, pid,
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) DeletePathologyTest(ctx context.Context, pool *pgxpool.Pool, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx, `
		DELETE FROM pathology_tests
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// EnsurePathologyParamCols self-heals the schema. Idempotent.
func (r *Repository) EnsurePathologyParamCols(ctx context.Context, pool *pgxpool.Pool) {
	_, _ = pool.Exec(ctx, `
		ALTER TABLE pathology_parameters
			ADD COLUMN IF NOT EXISTS units          TEXT  NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS ref_min        TEXT  NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS ref_max        TEXT  NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS method         TEXT  NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS patient_groups JSONB NOT NULL DEFAULT '[]'::jsonb,
			ADD COLUMN IF NOT EXISTS updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
	`)
	_, _ = pool.Exec(ctx, `
		DROP INDEX IF EXISTS pathology_parameters_clinic_name_idx;
		CREATE UNIQUE INDEX IF NOT EXISTS pathology_parameters_clinic_name_ux
			ON pathology_parameters (clinic_id, lower(parameter_name))
	`)
}

func (r *Repository) ListPathologyParameters(ctx context.Context, pool *pgxpool.Pool, testID, clinicID string) ([]model.PathologyParameterDTO, error) {
	r.EnsurePathologyParamCols(ctx, pool)
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
	}
	var err error
	if testID != "" {
		rows, err = pool.Query(ctx, `
			SELECT pp.id::text, pp.clinic_id::text, pp.parameter_name, pp.units,
			       pp.ref_min::text, pp.ref_max::text, COALESCE(pp.method, ''),
			       pp.patient_groups::text,
			       pp.created_at::text, pp.updated_at::text
			FROM pathology_parameters pp
			JOIN pathology_test_parameters ptp ON ptp.param_id = pp.id
			WHERE ptp.test_id = $1::uuid
			ORDER BY pp.parameter_name`,
			testID,
		)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT pp.id::text, pp.clinic_id::text, pp.parameter_name, pp.units,
			       pp.ref_min::text, pp.ref_max::text, COALESCE(pp.method, ''),
			       pp.patient_groups::text,
			       pp.created_at::text, pp.updated_at::text
			FROM pathology_parameters pp
			WHERE pp.clinic_id = $1::uuid
			ORDER BY pp.parameter_name`,
			clinicID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.PathologyParameterDTO{}
	for rows.Next() {
		var p model.PathologyParameterDTO
		var groupsStr string
		if err := rows.Scan(
			&p.ID, &p.TestID, &p.ParameterName, &p.Unit,
			&p.NormalRangeMin, &p.NormalRangeMax, &p.Method,
			&groupsStr,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if groupsStr != "" {
			_ = json.Unmarshal([]byte(groupsStr), &p.PatientGroups)
		}
		if p.PatientGroups == nil {
			p.PatientGroups = []model.PatientGroupRange{}
		}
		out = append(out, p)
	}
	return out, nil
}

func (r *Repository) DeriveClinicFromTest(ctx context.Context, pool *pgxpool.Pool, testID string) (string, error) {
	var clinicID string
	err := pool.QueryRow(ctx, "SELECT clinic_id::text FROM pathology_tests WHERE id = $1::uuid", testID).Scan(&clinicID)
	return clinicID, err
}

func (r *Repository) UpsertPathologyParameter(ctx context.Context, pool *pgxpool.Pool, clinicID string, req model.CreatePathologyParamRequest) (string, error) {
	r.EnsurePathologyParamCols(ctx, pool)
	groupsJSON, _ := json.Marshal(req.PatientGroups)
	var paramID string
	err := pool.QueryRow(ctx, `
		INSERT INTO pathology_parameters (clinic_id, parameter_name, units, ref_min, ref_max, method, patient_groups)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::jsonb)
		ON CONFLICT (clinic_id, lower(parameter_name)) DO UPDATE SET
			units = EXCLUDED.units,
			ref_min = EXCLUDED.ref_min,
			ref_max = EXCLUDED.ref_max,
			method = EXCLUDED.method,
			patient_groups = EXCLUDED.patient_groups,
			updated_at = now()
		RETURNING id::text`,
		clinicID, strings.TrimSpace(req.ParameterName), req.Unit,
		req.NormalRangeMin, req.NormalRangeMax, req.Method, string(groupsJSON),
	).Scan(&paramID)
	return paramID, err
}

func (r *Repository) LinkTestParameter(ctx context.Context, pool *pgxpool.Pool, testID, paramID string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO pathology_test_parameters (test_id, param_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT (test_id, param_id) DO NOTHING`,
		testID, paramID,
	)
	return err
}

func (r *Repository) UpdatePathologyParameter(ctx context.Context, pool *pgxpool.Pool, id, userID string, req model.CreatePathologyParamRequest) (int64, error) {
	r.EnsurePathologyParamCols(ctx, pool)
	groupsJSON, _ := json.Marshal(req.PatientGroups)
	ct, err := pool.Exec(ctx, `
		UPDATE pathology_parameters SET
			parameter_name = $3, units = $4, ref_min = $5, ref_max = $6,
			method = $7, patient_groups = $8::jsonb, updated_at = now()
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
		strings.TrimSpace(req.ParameterName), req.Unit,
		req.NormalRangeMin, req.NormalRangeMax, req.Method, string(groupsJSON),
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) DeletePathologyParameter(ctx context.Context, pool *pgxpool.Pool, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx, `
		DELETE FROM pathology_parameters
		WHERE id = $1::uuid
		  AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) ListPathologyGroups(ctx context.Context, pool *pgxpool.Pool, clinicID string) ([]model.ParamGroupDTO, error) {
	rows, err := pool.Query(ctx, `
		SELECT g.id::text, g.clinic_id::text, g.group_name, g.created_at::text,
		       COALESCE(array_agg(p.id::text ORDER BY p.parameter_name) FILTER (WHERE p.id IS NOT NULL), '{}'),
		       COALESCE(array_agg(p.parameter_name ORDER BY p.parameter_name) FILTER (WHERE p.id IS NOT NULL), '{}')
		FROM pathology_parameter_groups g
		LEFT JOIN pathology_group_parameters gp ON gp.group_id = g.id
		LEFT JOIN pathology_parameters p ON p.id = gp.param_id
		WHERE g.clinic_id = $1::uuid
		GROUP BY g.id, g.clinic_id, g.group_name, g.created_at
		ORDER BY g.group_name`,
		clinicID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.ParamGroupDTO{}
	for rows.Next() {
		var g model.ParamGroupDTO
		if err := rows.Scan(&g.ID, &g.ClinicID, &g.GroupName, &g.CreatedAt, &g.ParamIDs, &g.ParamNames); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (r *Repository) UpsertPathologyGroup(ctx context.Context, pool *pgxpool.Pool, clinicID, groupName string, paramIDs []string) (string, error) {
	var id string
	if err := pool.QueryRow(ctx,
		`INSERT INTO pathology_parameter_groups (clinic_id, group_name)
		 VALUES ($1::uuid, $2)
		 ON CONFLICT (clinic_id, lower(group_name)) DO UPDATE SET group_name = EXCLUDED.group_name
		 RETURNING id::text`,
		clinicID, strings.TrimSpace(groupName),
	).Scan(&id); err != nil {
		return "", err
	}
	_, _ = pool.Exec(ctx, `DELETE FROM pathology_group_parameters WHERE group_id = $1::uuid`, id)
	for _, pid := range paramIDs {
		if pid == "" {
			continue
		}
		_, _ = pool.Exec(ctx,
			`INSERT INTO pathology_group_parameters (group_id, param_id)
			 VALUES ($1::uuid, $2::uuid)
			 ON CONFLICT (group_id, param_id) DO NOTHING`,
			id, pid,
		)
	}
	return id, nil
}

func (r *Repository) UpdatePathologyGroup(ctx context.Context, pool *pgxpool.Pool, id, userID, groupName string, paramIDs []string) (int64, error) {
	ct, err := pool.Exec(ctx,
		`UPDATE pathology_parameter_groups
		 SET group_name = $2
		 WHERE id = $1::uuid
		   AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $3::uuid)`,
		id, strings.TrimSpace(groupName), userID,
	)
	if err != nil {
		return 0, err
	}
	if ct.RowsAffected() == 0 {
		return 0, nil
	}
	_, _ = pool.Exec(ctx, `DELETE FROM pathology_group_parameters WHERE group_id = $1::uuid`, id)
	for _, pid := range paramIDs {
		if pid == "" {
			continue
		}
		_, _ = pool.Exec(ctx,
			`INSERT INTO pathology_group_parameters (group_id, param_id)
			 VALUES ($1::uuid, $2::uuid)
			 ON CONFLICT (group_id, param_id) DO NOTHING`,
			id, pid,
		)
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) DeletePathologyGroup(ctx context.Context, pool *pgxpool.Pool, id, userID string) (int64, error) {
	ct, err := pool.Exec(ctx,
		`DELETE FROM pathology_parameter_groups
		 WHERE id = $1::uuid
		   AND clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $2::uuid)`,
		id, userID,
	)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

func (r *Repository) RemoveGroupParam(ctx context.Context, pool *pgxpool.Pool, groupID, paramID, userID string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM pathology_group_parameters
		 WHERE group_id = $1::uuid AND param_id = $2::uuid
		   AND group_id IN (SELECT id FROM pathology_parameter_groups
		                    WHERE clinic_id IN (SELECT clinic_id FROM clinic_members WHERE user_id = $3::uuid))`,
		groupID, paramID, userID,
	)
	return err
}
