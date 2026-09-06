// Package seeder contains idempotent startup seeders — the default super
// admin and the demo hospital tenant.
package seeder

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/utils"
	"github.com/salman/hms-backend/internal/infrastructure/persistence/postgres"
)

// EnsureSuperAdmin creates the default super admin when the table is empty.
func EnsureSuperAdmin(ctx context.Context, registry *pgxpool.Pool, email, password string) error {
	var count int
	if err := registry.QueryRow(ctx, `SELECT count(*) FROM super_admins`).Scan(&count); err != nil {
		return fmt.Errorf("count super admins: %w", err)
	}
	if count > 0 {
		return nil
	}
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = registry.Exec(ctx, `
		INSERT INTO super_admins (email, password_hash, full_name)
		VALUES ($1, $2, 'Platform Super Admin')`,
		email, hash,
	)
	if err != nil {
		return fmt.Errorf("insert super admin: %w", err)
	}
	return nil
}

// EnsureDemoTenant provisions the `demo` hospital on first run and seeds it
// with an admin user plus six clinics. Safe to call on every startup.
func EnsureDemoTenant(ctx context.Context, resolver *postgres.TenantResolver) error {
	info, pool, err := resolver.Provision(ctx, "demo", "Demo Hospital")
	if err != nil {
		return fmt.Errorf("provision demo: %w", err)
	}

	var hasUsers bool
	if err := resolver.Registry().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_directory WHERE tenant_id = $1::uuid)`, info.ID,
	).Scan(&hasUsers); err != nil {
		return err
	}
	if hasUsers {
		// still ensure sui@gmail.com exists even on subsequent startups
		return ensureSuiUser(ctx, resolver, pool, info.ID)
	}

	hash, err := utils.HashPassword("Passw0rd!")
	if err != nil {
		return err
	}

	var adminID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role, is_super_admin)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id::text`,
		"admin@demo.clinic", hash, "Demo Admin", "admin",
	).Scan(&adminID); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	if _, err := resolver.Registry().Exec(ctx,
		`INSERT INTO user_directory (email, tenant_id, user_id)
		 VALUES ($1, $2::uuid, $3::uuid)
		 ON CONFLICT (email) DO NOTHING`,
		"admin@demo.clinic", info.ID, adminID,
	); err != nil {
		return fmt.Errorf("register admin in directory: %w", err)
	}

	userHash, err := utils.HashPassword("Passw0rd!")
	if err != nil {
		return err
	}
	var userID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text`,
		"user@demo.clinic", userHash, "Demo User", "user",
	).Scan(&userID); err != nil {
		return fmt.Errorf("seed regular user: %w", err)
	}

	if _, err := resolver.Registry().Exec(ctx,
		`INSERT INTO user_directory (email, tenant_id, user_id)
		 VALUES ($1, $2::uuid, $3::uuid)
		 ON CONFLICT (email) DO NOTHING`,
		"user@demo.clinic", info.ID, userID,
	); err != nil {
		return fmt.Errorf("register regular user in directory: %w", err)
	}

	clinics := []struct {
		name, clinicType, location, color, code string
	}{
		{"HealthCare Plus", "Multi-Specialty Clinic", "New York, NY", "purple", "HCPLUS"},
		{"Support Center", "Counseling Center", "Chicago, IL", "blue", "SUPRT1"},
		{"Recovery Center", "Rehabilitation Facility", "Los Angeles, CA", "green", "RECOV1"},
		{"Family Center", "Support Services", "Miami, FL", "orange", "FAMLY1"},
		{"Healing Clinic", "Mental Health Facility", "Seattle, WA", "pink", "HEALNG"},
		{"Mindfulness Center", "Wellness Studio", "Austin, TX", "indigo", "MINDFL"},
	}
	clinicIDs := make([]string, 0, len(clinics))
	for i, s := range clinics {
		var cID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO clinics (name, clinic_type, location, color, invite_code)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id::text`,
			s.name, s.clinicType, s.location, s.color, s.code,
		).Scan(&cID); err != nil {
			return fmt.Errorf("seed clinic %q: %w", s.name, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO clinic_members (clinic_id, user_id, role, is_default)
			VALUES ($1::uuid, $2::uuid, 'admin', $3)`,
			cID, userID, i == 0,
		); err != nil {
			return fmt.Errorf("seed member for clinic %q: %w", s.name, err)
		}
		clinicIDs = append(clinicIDs, cID)
	}

	// seed inventory for first clinic
	if len(clinicIDs) > 0 {
		_ = seedInventory(ctx, pool, clinicIDs[0])
	}

	return ensureSuiUser(ctx, resolver, pool, info.ID)
}

// EnsureInventoryData seeds demo inventory for every clinic in every tenant
// that does not already have inventory categories. Idempotent.
func EnsureInventoryData(ctx context.Context, resolver *postgres.TenantResolver) error {
	tenants, err := resolver.AllTenantIDs(ctx)
	if err != nil {
		return nil // not fatal
	}
	for _, tid := range tenants {
		pool, err := resolver.Pool(ctx, tid)
		if err != nil {
			continue
		}
		rows, err := pool.Query(ctx, `SELECT id FROM clinics`)
		if err != nil {
			continue
		}
		var ids []string
		for rows.Next() {
			var cid string
			if rows.Scan(&cid) == nil {
				ids = append(ids, cid)
			}
		}
		rows.Close()
		for _, cid := range ids {
			_ = seedInventory(ctx, pool, cid)
		}
	}
	return nil
}

// ensureSuiUser creates sui@gmail.com (password = "sui@gmail.com") and adds
// them to all demo clinics as admin. Idempotent.
func ensureSuiUser(ctx context.Context, resolver *postgres.TenantResolver, pool *pgxpool.Pool, tenantID string) error {
	const suiEmail = "sui@gmail.com"
	const suiPass = "sui@gmail.com"

	var exists bool
	_ = resolver.Registry().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_directory WHERE email=$1)`, suiEmail,
	).Scan(&exists)
	if exists {
		return nil
	}

	suiHash, err := utils.HashPassword(suiPass)
	if err != nil {
		return err
	}
	var suiID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name, role, is_super_admin)
		VALUES ($1, $2, 'Sui Admin', 'admin', true)
		ON CONFLICT (email) DO UPDATE SET password_hash=EXCLUDED.password_hash
		RETURNING id::text`, suiEmail, suiHash,
	).Scan(&suiID); err != nil {
		return fmt.Errorf("upsert sui user: %w", err)
	}

	if _, err := resolver.Registry().Exec(ctx,
		`INSERT INTO user_directory (email, tenant_id, user_id)
		 VALUES ($1, $2::uuid, $3::uuid) ON CONFLICT (email) DO NOTHING`,
		suiEmail, tenantID, suiID,
	); err != nil {
		return fmt.Errorf("register sui in directory: %w", err)
	}

	// add to all clinics in this tenant
	rows, err := pool.Query(ctx, `SELECT id FROM clinics LIMIT 20`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	clinicIDs := []string{}
	first := true
	for rows.Next() {
		var cID string
		if err := rows.Scan(&cID); err != nil {
			continue
		}
		_, _ = pool.Exec(ctx, `
			INSERT INTO clinic_members (clinic_id, user_id, role, is_default)
			VALUES ($1::uuid, $2::uuid, 'admin', $3)
			ON CONFLICT DO NOTHING`, cID, suiID, first)
		clinicIDs = append(clinicIDs, cID)
		first = false
	}
	// seed demo pathology tests + parameters for every clinic (idempotent)
	for _, cid := range clinicIDs {
		_ = seedPathology(ctx, pool, cid)
	}
	return nil
}

// seedPathology inserts a small but realistic set of pathology tests with
// parameters for a given clinic. Idempotent — skips if tests already exist.
func seedPathology(ctx context.Context, pool *pgxpool.Pool, clinicID string) error {
	var already bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pathology_tests WHERE clinic_id=$1::uuid)`, clinicID).Scan(&already)
	if already {
		return nil
	}

	// categories
	catNames := []string{"Hematology", "Biochemistry", "Endocrinology", "Serology"}
	for _, n := range catNames {
		_, _ = pool.Exec(ctx, `
			INSERT INTO pathology_categories (clinic_id, name) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, clinicID, n)
	}

	// tests + parameter specs
	type paramSpec struct {
		name, unit, min, max, method string
		groups                       string // JSON array
	}
	type testSpec struct {
		name, code, category, sample, tax string
		price                             float64
		params                            []paramSpec
	}
	tests := []testSpec{
		{
			"Complete Blood Count (CBC)", "CBC001", "Hematology", "Whole Blood", "GST (5%)", 450,
			[]paramSpec{
				{"Hemoglobin", "g/dL", "12.0", "17.5", "Colorimetric",
					`[{"patientCategory":"Male","low":"13.5","high":"17.5"},{"patientCategory":"Female","low":"12.0","high":"15.5"},{"patientCategory":"Infant","low":"10.0","high":"14.0"}]`},
				{"RBC Count", "10^12/L", "4.5", "5.9", "Automated", `[]`},
				{"WBC Count", "10^9/L", "4.0", "11.0", "Automated", `[]`},
				{"Platelet Count", "10^9/L", "150", "450", "Automated", `[]`},
				{"Hematocrit", "%", "36", "50", "Automated",
					`[{"patientCategory":"Male","low":"41","high":"50"},{"patientCategory":"Female","low":"36","high":"44"}]`},
			},
		},
		{
			"Liver Function Test (LFT)", "LFT001", "Biochemistry", "Serum", "GST (5%)", 650,
			[]paramSpec{
				{"ALT (SGPT)", "U/L", "7", "56", "IFCC", `[]`},
				{"AST (SGOT)", "U/L", "10", "40", "IFCC", `[]`},
				{"Total Bilirubin", "mg/dL", "0.1", "1.2", "Diazo", `[]`},
				{"Albumin", "g/dL", "3.5", "5.0", "BCG", `[]`},
			},
		},
		{
			"Kidney Function Test (KFT)", "KFT001", "Biochemistry", "Serum", "GST (5%)", 550,
			[]paramSpec{
				{"Urea", "mg/dL", "10", "40", "Urease", `[]`},
				{"Creatinine", "mg/dL", "0.6", "1.3", "Jaffe",
					`[{"patientCategory":"Male","low":"0.7","high":"1.3"},{"patientCategory":"Female","low":"0.6","high":"1.1"}]`},
				{"Uric Acid", "mg/dL", "3.5", "7.2", "Uricase", `[]`},
			},
		},
		{
			"Thyroid Profile", "TFT001", "Endocrinology", "Serum", "GST (5%)", 750,
			[]paramSpec{
				{"TSH", "mIU/L", "0.4", "4.5", "CLIA",
					`[{"patientCategory":"Pregnant","low":"0.1","high":"2.5"}]`},
				{"Free T3", "pg/mL", "2.3", "4.2", "CLIA", `[]`},
				{"Free T4", "ng/dL", "0.8", "1.8", "CLIA", `[]`},
			},
		},
		{
			"Lipid Profile", "LIP001", "Biochemistry", "Serum", "GST (5%)", 600,
			[]paramSpec{
				{"Total Cholesterol", "mg/dL", "", "200", "CHOD-PAP", `[]`},
				{"HDL", "mg/dL", "40", "60", "Direct",
					`[{"patientCategory":"Male","low":"40","high":"60"},{"patientCategory":"Female","low":"50","high":"70"}]`},
				{"LDL", "mg/dL", "", "100", "Calculated", `[]`},
				{"Triglycerides", "mg/dL", "", "150", "GPO-PAP", `[]`},
			},
		},
	}

	for _, t := range tests {
		var testID string
		if err := pool.QueryRow(ctx, `
			INSERT INTO pathology_tests (clinic_id, test_name, test_code, category, test_type, price, tax)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
			RETURNING id::text`,
			clinicID, t.name, t.code, t.category, t.sample, t.price, t.tax,
		).Scan(&testID); err != nil {
			continue
		}
		for _, p := range t.params {
			var paramID string
			if err := pool.QueryRow(ctx, `
				INSERT INTO pathology_parameters (clinic_id, parameter_name, units, ref_min, ref_max, method, patient_groups)
				VALUES ($1::uuid, $2, $3, $4, $5, $6, $7::jsonb)
				ON CONFLICT (clinic_id, lower(parameter_name)) DO UPDATE SET
					units = EXCLUDED.units, ref_min = EXCLUDED.ref_min,
					ref_max = EXCLUDED.ref_max, method = EXCLUDED.method,
					patient_groups = EXCLUDED.patient_groups
				RETURNING id::text`,
				clinicID, p.name, p.unit, p.min, p.max, p.method, p.groups,
			).Scan(&paramID); err != nil {
				continue
			}
			_, _ = pool.Exec(ctx, `
				INSERT INTO pathology_test_parameters (test_id, param_id)
				VALUES ($1::uuid, $2::uuid) ON CONFLICT DO NOTHING`,
				testID, paramID)
		}
	}
	return nil
}

// seedInventory inserts demo drugs, categories, manufacturers, stock, sales,
// dispenses, assets and purchases for a given clinic. Idempotent.
func seedInventory(ctx context.Context, pool *pgxpool.Pool, clinicID string) error {
	var already bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM inv_categories WHERE clinic_id=$1::uuid)`, clinicID).Scan(&already)
	if already {
		return nil
	}

	// categories
	cats := []string{"Antibiotics", "Analgesics", "Antidiabetics", "Vitamins", "Antihypertensives", "Antacids"}
	catIDs := make([]string, len(cats))
	for i, name := range cats {
		_ = pool.QueryRow(ctx, `INSERT INTO inv_categories (clinic_id, name) VALUES ($1::uuid,$2) RETURNING id`, clinicID, name).Scan(&catIDs[i])
	}

	// manufacturers
	mfrs := []string{"Sun Pharma", "Cipla Ltd", "Dr. Reddy's", "Abbott India", "Pfizer India", "Lupin Ltd"}
	mfrIDs := make([]string, len(mfrs))
	for i, name := range mfrs {
		_ = pool.QueryRow(ctx, `INSERT INTO inv_manufacturers (clinic_id, name) VALUES ($1::uuid,$2) RETURNING id`, clinicID, name).Scan(&mfrIDs[i])
	}

	// drugs + stock
	type drugSeed struct {
		name, generic, strength, code string
		catIdx, mfrIdx                int
		qty                           int
		mrp, prate                    float64
	}
	drugs := []drugSeed{
		{"Amoxicillin 500mg", "Amoxicillin", "500mg", "AMX001", 0, 0, 121, 12.50, 8.00},
		{"Paracetamol 650mg", "Paracetamol", "650mg", "PCM001", 1, 1, 450, 3.75, 2.00},
		{"Omeprazole 20mg", "Omeprazole", "20mg", "OMP001", 5, 2, 200, 8.25, 5.50},
		{"Metformin 500mg", "Metformin", "500mg", "MET001", 2, 3, 180, 5.00, 3.00},
		{"Amlodipine 5mg", "Amlodipine", "5mg", "AML001", 4, 4, 95, 6.50, 4.00},
		{"Vitamin C 500mg", "Ascorbic Acid", "500mg", "VTC001", 3, 5, 300, 4.00, 2.50},
		{"Azithromycin 500mg", "Azithromycin", "500mg", "AZT001", 0, 0, 60, 18.00, 12.00},
		{"Insulin Glargine", "Insulin Glargine", "100U/mL", "INS001", 2, 1, 8, 950.00, 700.00},
		{"Ibuprofen 400mg", "Ibuprofen", "400mg", "IBP001", 1, 2, 320, 5.50, 3.50},
		{"Atorvastatin 10mg", "Atorvastatin", "10mg", "ATV001", 4, 3, 150, 9.00, 6.00},
		{"Cetirizine 10mg", "Cetirizine", "10mg", "CTZ001", 1, 4, 200, 4.50, 2.80},
		{"Pantoprazole 40mg", "Pantoprazole", "40mg", "PNT001", 5, 5, 160, 7.50, 4.80},
		{"Ciprofloxacin 500mg", "Ciprofloxacin", "500mg", "CIP001", 0, 1, 80, 14.00, 9.00},
		{"Dolo 650", "Paracetamol", "650mg", "DLO001", 1, 0, 500, 3.50, 1.80},
		{"Syringes 5ml (Pack)", "Disposable Syringe", "5ml", "SYR001", 1, 2, 200, 12.00, 7.00},
	}

	drugIDs := make([]string, len(drugs))
	for i, d := range drugs {
		var catID, mfrID *string
		if d.catIdx < len(catIDs) && catIDs[d.catIdx] != "" {
			catID = &catIDs[d.catIdx]
		}
		if d.mfrIdx < len(mfrIDs) && mfrIDs[d.mfrIdx] != "" {
			mfrID = &mfrIDs[d.mfrIdx]
		}
		_ = pool.QueryRow(ctx, `
			INSERT INTO inv_drugs (clinic_id, name, generic_name, category_id, strength, item_code, manufacturer_id)
			VALUES ($1::uuid,$2,$3,$4::uuid,$5,$6,$7::uuid) RETURNING id`,
			clinicID, d.name, d.generic, catID, d.strength, d.code, mfrID,
		).Scan(&drugIDs[i])

		if drugIDs[i] != "" {
			_, _ = pool.Exec(ctx, `
				INSERT INTO inv_stock (clinic_id, drug_id, batch_code, expiry_date, qty_available, purchase_rate, mrp)
				VALUES ($1::uuid,$2::uuid,$3,$4::date,$5,$6,$7)`,
				clinicID, drugIDs[i],
				fmt.Sprintf("BCH-%04d", i+1),
				fmt.Sprintf("2027-%02d-01", (i%12)+1),
				d.qty, d.prate, d.mrp,
			)
		}
	}

	// sales
	saleCustomers := []string{"John Smith", "Jane Doe", "Alice Johnson", "Bob Williams", "Carol Martinez", "David Lee"}
	for i, cust := range saleCustomers {
		invoiceNo := fmt.Sprintf("INV-2024-%04d", i+1)
		var saleID string
		err := pool.QueryRow(ctx, `
			INSERT INTO inv_sales (clinic_id, invoice_no, customer_name, payment_method, subtotal, grand_total, status, created_at)
			VALUES ($1::uuid,$2,$3,'Cash',$4,$4,'Paid', NOW() - ($5 * interval '1 day'))
			RETURNING id`,
			clinicID, invoiceNo, cust, float64(i+1)*19.80, i,
		).Scan(&saleID)
		if err == nil && saleID != "" && i < len(drugIDs) && drugIDs[i] != "" {
			_, _ = pool.Exec(ctx, `
				INSERT INTO inv_sale_items (sale_id, drug_id, drug_name, unit, qty, unit_price, total)
				VALUES ($1::uuid,$2::uuid,$3,'Tablet',2,$4,$4*2)`,
				saleID, drugIDs[i], drugs[i].name, drugs[i].mrp,
			)
		}
	}

	// dispenses
	patients := []string{"Ravi Kumar", "Priya Sharma", "Arjun Nair", "Meena Patel", "Suresh Iyer", "Kavya Reddy"}
	depts := []string{"General Ward", "ICU", "OPD", "Emergency", "Surgery", "General Ward"}
	for i, pat := range patients {
		dispNo := fmt.Sprintf("DSP-%d-%03d", 2024, i+1)
		var dispID string
		err := pool.QueryRow(ctx, `
			INSERT INTO inv_dispenses (clinic_id, dispense_no, patient_name, issued_to_dept, prescribed_by, issued_date, status, created_at)
			VALUES ($1::uuid,$2,$3,$4,'Dr. Sarah Johnson',$5::date,'Dispensed', NOW() - ($6 * interval '1 day'))
			RETURNING id`,
			clinicID, dispNo, pat, depts[i],
			fmt.Sprintf("2024-%02d-15", (i%12)+1),
			i,
		).Scan(&dispID)
		if err == nil && dispID != "" && i < len(drugIDs) && drugIDs[i] != "" {
			_, _ = pool.Exec(ctx, `
				INSERT INTO inv_dispense_items (dispense_id, drug_id, drug_name, unit, qty_prescribed, qty_total)
				VALUES ($1::uuid,$2::uuid,$3,'Tablet',12,12)`,
				dispID, drugIDs[i], drugs[i].name,
			)
		}
	}

	// assets
	type assetSeed struct {
		name, mfr, serial, model, dept, location, category, condition, status string
		cost, value                                                           float64
	}
	assets := []assetSeed{
		{"Adjustable Hospital Beds (Set of 10)", "Progress Bed", "SN-BED-001", "HH-Item", "General Ward", "Ward A", "Furniture", "Good", "Active", 14000, 9500},
		{"ECG Machine", "Philips", "SN-ECG-001", "PageWriter TC30", "ICU", "ICU Room 1", "Medical", "Good", "Active", 85000, 62000},
		{"Ventilator", "Drager", "SN-VEN-001", "Evita Infinity V500", "ICU", "ICU Room 2", "Medical", "Good", "Active", 350000, 280000},
		{"X-Ray Machine", "Siemens", "SN-XRY-001", "Multix Select", "Radiology", "Radiology Dept", "Medical", "Good", "Active", 420000, 310000},
		{"Ultrasound Scanner", "GE Healthcare", "SN-ULT-001", "LOGIQ P9", "OPD", "OPD Room 3", "Medical", "Good", "Active", 180000, 140000},
		{"Autoclave Sterilizer", "Tuttnauer", "SN-AUT-001", "2540M", "Surgery", "OT Room 1", "Medical", "Fair", "Maintenance", 45000, 32000},
		{"Patient Monitor", "Mindray", "SN-MON-001", "MEC-1000", "ICU", "ICU Room 3", "Medical", "Good", "Active", 35000, 28000},
		{"Infusion Pump", "BD Alaris", "SN-INF-001", "8015FC", "General Ward", "Ward B", "Medical", "Good", "Active", 18000, 14000},
		{"Wheelchair (Set of 5)", "Karma Medical", "SN-WCH-001", "S-Ergo", "OPD", "Corridor A", "Equipment", "Good", "Active", 12000, 9000},
		{"Examination Table", "Clinton Industries", "SN-TBL-001", "8-Section", "OPD", "OPD Room 1", "Furniture", "Good", "Active", 22000, 18000},
		{"Defibrillator", "Zoll Medical", "SN-DEF-001", "AED Plus", "Emergency", "Emergency Bay", "Medical", "Good", "Active", 95000, 72000},
		{"Oxygen Concentrator", "Philips Respironics", "SN-OXY-001", "SimplyFlo", "ICU", "ICU Room 1", "Medical", "Fair", "Maintenance", 28000, 19000},
	}
	for i, a := range assets {
		assetNo := fmt.Sprintf("AST-PUR-%03d", i+1)
		_, _ = pool.Exec(ctx, `
			INSERT INTO inv_assets (clinic_id, asset_no, name, manufacturer, serial_number, model, location,
			    purchase_date, current_value, purchase_cost, category, department, condition, status)
			VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8::date,$9,$10,$11,$12,$13,$14)`,
			clinicID, assetNo, a.name, a.mfr, a.serial, a.model, a.location,
			fmt.Sprintf("2022-%02d-01", (i%12)+1),
			a.value, a.cost, a.category, a.dept, a.condition, a.status,
		)
	}

	// purchases
	suppliers := []string{"MedSupply Co.", "PharmaDirect India", "HealthBridge Pvt Ltd"}
	for i, sup := range suppliers {
		purID := fmt.Sprintf("PUR-%04d", i+1)
		var purchaseID string
		err := pool.QueryRow(ctx, `
			INSERT INTO inv_purchases (clinic_id, purchase_id, invoice_no, supplier_name, payment_method, subtotal, grand_total, created_at)
			VALUES ($1::uuid,$2,$3,$4,'Cash',$5,$5, NOW() - ($6 * interval '7 days'))
			RETURNING id`,
			clinicID, purID,
			fmt.Sprintf("SINV-%04d", i+100),
			sup,
			float64(i+1)*1240.00,
			i,
		).Scan(&purchaseID)
		if err == nil && purchaseID != "" && i < len(drugIDs) && drugIDs[i] != "" {
			_, _ = pool.Exec(ctx, `
				INSERT INTO inv_purchase_items (purchase_id, drug_id, drug_name, batch_code, qty_purchased, qty_total, purchase_rate, mrp, amount)
				VALUES ($1::uuid,$2::uuid,$3,$4,100,100,$5,$6,$7)`,
				purchaseID, drugIDs[i], drugs[i].name,
				fmt.Sprintf("BCH-%04d", i+50),
				drugs[i].prate, drugs[i].mrp,
				drugs[i].prate*100,
			)
		}
	}

	return nil
}
