package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/modules/catalogue/model"
)

func (r *Repository) CopyCatalogue(ctx context.Context, pool *pgxpool.Pool, sourceID, targetID string) (model.CopyCatalogueResult, error) {
	res := model.CopyCatalogueResult{ClinicID: targetID}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer tx.Rollback(ctx)

	if ct, err := tx.Exec(ctx, `
		INSERT INTO drug_categories (clinic_id, name)
		SELECT $2::uuid, name FROM drug_categories WHERE clinic_id = $1::uuid
		ON CONFLICT (clinic_id, lower(name)) DO NOTHING`,
		sourceID, targetID,
	); err == nil {
		res.Categories = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if ct, err := tx.Exec(ctx, `
		INSERT INTO drug_manufacturers (clinic_id, name)
		SELECT $2::uuid, name FROM drug_manufacturers WHERE clinic_id = $1::uuid
		ON CONFLICT (clinic_id, lower(name)) DO NOTHING`,
		sourceID, targetID,
	); err == nil {
		res.Manufacturers = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if ct, err := tx.Exec(ctx, `
		INSERT INTO drug_units (clinic_id, name, kind)
		SELECT $2::uuid, name, kind FROM drug_units WHERE clinic_id = $1::uuid
		ON CONFLICT (clinic_id, kind, lower(name)) DO NOTHING`,
		sourceID, targetID,
	); err == nil {
		res.Units = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if ct, err := tx.Exec(ctx, `
		INSERT INTO drugs (clinic_id, drug_name, generic_name, category, strength, item_code,
		                   manufacturer, instruction, primary_unit, secondary_unit,
		                   reorder_level, hsn_code, tax, discount)
		SELECT $2::uuid, drug_name, generic_name, category, strength, item_code,
		       manufacturer, instruction, primary_unit, secondary_unit,
		       reorder_level, hsn_code, tax, discount
		FROM drugs src
		WHERE src.clinic_id = $1::uuid
		  AND NOT EXISTS (
		      SELECT 1 FROM drugs dst
		      WHERE dst.clinic_id = $2::uuid
		        AND lower(dst.drug_name) = lower(src.drug_name)
		        AND lower(dst.strength)  = lower(src.strength)
		  )`,
		sourceID, targetID,
	); err == nil {
		res.Drugs = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if ct, err := tx.Exec(ctx, `
		INSERT INTO radiology_categories (clinic_id, name)
		SELECT $2::uuid, name FROM radiology_categories WHERE clinic_id = $1::uuid
		ON CONFLICT (clinic_id, lower(name)) DO NOTHING`,
		sourceID, targetID,
	); err == nil {
		res.RadiologyCategories = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if ct, err := tx.Exec(ctx, `
		INSERT INTO radiology_tests (clinic_id, test_name, test_code, category, body_part,
		                             description, price, tax)
		SELECT $2::uuid, test_name, test_code, category, body_part, description, price, tax
		FROM radiology_tests src
		WHERE src.clinic_id = $1::uuid
		  AND NOT EXISTS (
		      SELECT 1 FROM radiology_tests dst
		      WHERE dst.clinic_id = $2::uuid
		        AND lower(dst.test_name) = lower(src.test_name)
		  )`,
		sourceID, targetID,
	); err == nil {
		res.RadiologyTests = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if ct, err := tx.Exec(ctx, `
		INSERT INTO treatments (clinic_id, treatment_name, treatment_code, description,
		                        price, discount, tax, consumables)
		SELECT $2::uuid, treatment_name, treatment_code, description,
		       price, discount, tax, consumables
		FROM treatments src
		WHERE src.clinic_id = $1::uuid
		  AND NOT EXISTS (
		      SELECT 1 FROM treatments dst
		      WHERE dst.clinic_id = $2::uuid
		        AND lower(dst.treatment_name) = lower(src.treatment_name)
		  )`,
		sourceID, targetID,
	); err == nil {
		res.Treatments = int(ct.RowsAffected())
	} else {
		return res, err
	}

	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	return res, nil
}
