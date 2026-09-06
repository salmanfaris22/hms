package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (r *Repository) SeedDemo(ctx context.Context, pool *pgxpool.Pool, patientID, clinicID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	exec := func(sql string, args ...any) error {
		_, err := tx.Exec(ctx, sql, args...)
		return err
	}

	_ = exec(`DELETE FROM patient_alerts         WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM patient_allergy_entries WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM patient_medications    WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM patient_vitals         WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM appointments           WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM patient_treatments     WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM invoices               WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM insurance_claims       WHERE patient_id = $1::uuid`, patientID)
	_ = exec(`DELETE FROM patient_events         WHERE patient_id = $1::uuid`, patientID)

	alerts := []struct{ name, sev string }{
		{"Hypertension", "high"},
		{"Diabetes (Type 2)", "medium"},
		{"Anxiety Disorder", "low"},
	}
	for _, a := range alerts {
		_ = exec(`INSERT INTO patient_alerts (patient_id, name, severity) VALUES ($1::uuid, $2, $3)`,
			patientID, a.name, a.sev)
	}

	allergies := []struct{ name, sev, reaction string }{
		{"Penicillin", "high", "Rash, difficulty breathing"},
		{"Latex", "medium", "Skin irritation"},
		{"Aspirin", "medium", "Stomach upset"},
		{"Shellfish", "low", "Itching"},
	}
	for _, a := range allergies {
		_ = exec(`INSERT INTO patient_allergy_entries (patient_id, name, severity, reaction) VALUES ($1::uuid, $2, $3, $4)`,
			patientID, a.name, a.sev, a.reaction)
	}

	meds := []struct{ name, dose, sched string }{
		{"Lisinopril", "10mg", "Once daily"},
		{"Aspirin", "81mg", "Once daily"},
		{"Atorvastatin", "20mg", "Once daily"},
	}
	for _, m := range meds {
		_ = exec(`INSERT INTO patient_medications (patient_id, name, dose, schedule, status, started_at)
		          VALUES ($1::uuid, $2, $3, $4, 'active', now() - INTERVAL '6 months')`,
			patientID, m.name, m.dose, m.sched)
	}

	vitals := []struct{ cat, kind, val, unit, status, refRange string }{
		{"body", "bmi", "23.4", "", "normal", "18.5 – 24.9"},
		{"body", "weight", "75", "kg", "normal", ""},
		{"body", "height", "175", "cm", "normal", ""},
		{"body", "temperature", "98.6", "°F", "normal", "97 – 99 °F"},
		{"blood", "blood_sugar", "98", "mg/dL", "normal", "70 – 140 mg/dL"},
		{"blood", "hemoglobin", "14.2", "g/dL", "normal", "13.5 – 17.5 g/dL"},
		{"blood", "platelets", "250", "K/µL", "normal", "150 – 400 K/µL"},
		{"heart", "blood_pressure", "120/80", "mmHg", "normal", "90/60 – 120/80 mmHg"},
		{"heart", "heart_rate", "72", "bpm", "normal", "60 – 100 bpm"},
		{"heart", "cholesterol", "180", "mg/dL", "normal", "< 200 mg/dL"},
		{"brain", "stress_index", "32", "/100", "normal", "0 – 40 /100"},
		{"brain", "sleep_quality", "78", "/100", "normal", "70 – 100 /100"},
		{"lungs", "respiratory_rate", "16", "bpm", "normal", "12 – 20 bpm"},
		{"lungs", "o2_level", "98", "%", "normal", "95 – 100 %"},
		{"lungs", "peak_flow", "550", "L/min", "normal", "400 – 700 L/min"},
		{"kidney", "creatinine", "1.0", "mg/dL", "normal", "0.7 – 1.3 mg/dL"},
		{"kidney", "gfr", "92", "mL/min", "normal", "> 90 mL/min"},
		{"dental", "total_teeth", "28", "", "normal", ""},
		{"dental", "cavities", "2", "", "elevated", ""},
		{"dental", "fillings", "4", "", "normal", ""},
		{"eye", "vision_left", "20/20", "", "normal", ""},
		{"eye", "vision_right", "20/25", "", "normal", ""},
		{"eye", "iop", "15", "mmHg", "normal", "10 – 21 mmHg"},
		{"skin", "condition", "Clear", "", "normal", ""},
		{"skin", "moles_tracked", "6", "", "normal", ""},
	}
	for _, v := range vitals {
		_ = exec(`INSERT INTO patient_vitals (patient_id, category, kind, value_text, unit, status, reference_range)
		          VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)`,
			patientID, v.cat, v.kind, v.val, v.unit, v.status, v.refRange)
	}

	trendPts := []int{92, 95, 99, 96, 101, 105, 110, 108, 102, 97, 94, 98}
	for i, v := range trendPts {
		when := time.Now().AddDate(0, -(11 - i), 0)
		_ = exec(`INSERT INTO patient_vitals (patient_id, recorded_at, category, kind, value_text, unit, status)
		          VALUES ($1::uuid, $2, 'blood', 'blood_sugar_trend', $3, 'mg/dL', 'normal')`,
			patientID, when, fmt.Sprintf("%d", v))
	}
	bpPts := []int{118, 121, 119, 123, 120, 124, 122, 120, 118, 121, 119, 120}
	for i, v := range bpPts {
		when := time.Now().AddDate(0, -(11 - i), 0)
		_ = exec(`INSERT INTO patient_vitals (patient_id, recorded_at, category, kind, value_text, unit, status)
		          VALUES ($1::uuid, $2, 'heart', 'blood_pressure_trend', $3, 'mmHg', 'normal')`,
			patientID, when, fmt.Sprintf("%d", v))
	}

	next := time.Now().AddDate(0, 0, 3).Truncate(24 * time.Hour).Add(10 * time.Hour)
	_ = exec(`INSERT INTO appointments (clinic_id, patient_id, provider_name, kind, scheduled_at, duration_min, status, notes)
	          VALUES ($1::uuid, $2::uuid, 'Dr. Albert Smith', 'General Consultation', $3, 30, 'scheduled', 'Follow-up visit')`,
		clinicID, patientID, next)

	for i, daysAgo := range []int{30, 120, 240} {
		when := time.Now().AddDate(0, 0, -daysAgo)
		_ = exec(`INSERT INTO appointments (clinic_id, patient_id, provider_name, kind, scheduled_at, duration_min, status, notes)
		          VALUES ($1::uuid, $2::uuid, 'Dr. Albert Smith', $3, $4, 30, 'completed', '')`,
			clinicID, patientID, []string{"Annual Check-up", "Follow-up Visit", "Lab Review"}[i], when)
	}

	treatments := []struct {
		name, cat string
		cents     int
		daysAgo   int
	}{
		{"Composite Filling", "Restorative", 18000, 45},
		{"Deep Cleaning (SRP)", "Periodontal", 22000, 90},
		{"Porcelain Crown", "Prosthetic", 98000, 150},
		{"X-Ray (Full Mouth)", "Diagnostic", 9000, 180},
		{"Root Canal Therapy", "Endodontic", 120000, 210},
	}
	for _, t := range treatments {
		when := time.Now().AddDate(0, 0, -t.daysAgo)
		_ = exec(`INSERT INTO patient_treatments (patient_id, name, category, amount_cents, performed_at)
		          VALUES ($1::uuid, $2, $3, $4, $5)`,
			patientID, t.name, t.cat, t.cents, when)
	}

	invoices := []struct {
		num                      string
		total, paid, due         int
		status                   string
		issuedDaysAgo, dueInDays int
	}{
		{"INV-1045", 25000, 25000, 0, "paid", 90, -60},
		{"INV-1078", 18000, 9000, 9000, "pending", 30, 10},
		{"INV-1092", 12000, 0, 12000, "overdue", 60, -15},
	}
	for _, inv := range invoices {
		issued := time.Now().AddDate(0, 0, -inv.issuedDaysAgo)
		due := time.Now().AddDate(0, 0, inv.dueInDays)
		_ = exec(`INSERT INTO invoices (clinic_id, patient_id, number, amount_total_cents, amount_paid_cents, amount_due_cents, status, issued_at, due_at, line_items)
		          VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::jsonb)`,
			clinicID, patientID, inv.num, inv.total, inv.paid, inv.due, inv.status, issued, due,
			fmt.Sprintf(`[{"name":"Consultation","amount":%d}]`, inv.total),
		)
	}

	_ = exec(`INSERT INTO insurance_claims (patient_id, provider, policy_number, claim_number, amount_claimed_cents, amount_approved_cents, amount_due_cents, status, submitted_at, resolved_at)
	          VALUES ($1::uuid, 'Blue Cross Blue Shield', 'BCBS-9981', 'CLM-442', 18000, 15000, 3000, 'approved', now() - INTERVAL '45 days', now() - INTERVAL '10 days')`,
		patientID)

	events := []struct {
		kind, title, desc string
		amount            int
		daysAgo           int
	}{
		{"visit", "Annual Check-up", "Completed with Dr. Smith", 0, 365},
		{"lab", "Lipid Panel", "Total cholesterol 180 mg/dL — normal", 0, 200},
		{"prescription", "Prescription Updated", "Started Lisinopril 10mg daily", 0, 180},
		{"emr", "Record Updated", "Added Type 2 Diabetes diagnosis", 0, 120},
		{"invoice", "Invoice INV-1045 issued", "Annual physical and lab work", 25000, 90},
		{"payment", "Payment Received", "$250 paid toward INV-1045", 25000, 88},
		{"insurance_claim", "Insurance Claim Approved", "Blue Cross reimbursed $150", 15000, 45},
		{"note", "Care Note", "Patient reports feeling better on new medication", 0, 14},
		{"visit", "Follow-up Visit", "BP reading 120/80, within target", 0, 3},
	}
	for _, e := range events {
		when := time.Now().AddDate(0, 0, -e.daysAgo)
		var amt any
		if e.amount > 0 {
			amt = e.amount
		}
		_ = exec(`INSERT INTO patient_events (patient_id, kind, title, description, amount_cents, occurred_at)
		          VALUES ($1::uuid, $2, $3, $4, $5, $6)`,
			patientID, e.kind, e.title, e.desc, amt, when)
	}

	return tx.Commit(ctx)
}
