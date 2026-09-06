package service

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/salman/hms-backend/internal/core/audit"
	"github.com/salman/hms-backend/internal/modules/patient/model"
	"github.com/salman/hms-backend/pkg/messaging"
)

// SettingsReader is the slice of the clinic repository this package needs. Kept
// as an interface so the patient module does not depend on the clinic module.
type SettingsReader interface {
	GetJSONSettings(ctx context.Context, pool *pgxpool.Pool, table, clinicID string) ([]byte, error)
}

// channelConfig mirrors one entry of the communication_settings blob written by
// the Message Configuration screen.
type channelConfig struct {
	SMS      bool   `json:"sms"`
	WhatsApp bool   `json:"whatsapp"`
	Email    bool   `json:"email"`
	AutoSend bool   `json:"autoSend"`
	Link     string `json:"link"`
}

type commsSettings struct {
	Messages map[string]channelConfig `json:"messages"`
}

// notifyWelcome dispatches the Welcome New Patient message on whichever channels
// the clinic enabled. It runs on its own context so a slow or failing provider
// can never delay or fail patient registration.
func (s *Service) notifyWelcome(tenantID, clinicID, patientNumber string, req model.CreateRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cfg, ok := s.messageConfig(ctx, tenantID, clinicID, messaging.KeyWelcomeNewPatient)
	if !ok || !cfg.AutoSend {
		return
	}
	tmpl, ok := messaging.Get(messaging.KeyWelcomeNewPatient)
	if !ok {
		return
	}

	name := strings.TrimSpace(req.FirstName + " " + req.LastName)
	if name == "" {
		name = "there"
	}
	vars := map[string]string{
		"patientName":   name,
		"patientNumber": patientNumber,
		"clinicName":    s.clinicName(ctx, tenantID, clinicID),
		"link":          cfg.Link,
	}

	phone := primaryPhone(req.Phones)

	if cfg.Email && req.Email != "" {
		subject := messaging.Render(tmpl.EmailSubject, vars)
		body := withLink(messaging.Render(tmpl.EmailBody, vars), cfg.Link)
		s.send(ctx, tenantID, "email", req.Email, patientNumber, func() error {
			return s.mailer.Send(ctx, req.Email, subject, messaging.TextToHTML(body))
		})
	}
	if cfg.SMS && phone != "" {
		body := withLink(messaging.Render(tmpl.SMS, vars), cfg.Link)
		s.send(ctx, tenantID, "sms", phone, patientNumber, func() error {
			return s.sms.Send(ctx, phone, body)
		})
	}
	if cfg.WhatsApp && phone != "" {
		body := withLink(messaging.Render(tmpl.WhatsApp, vars), cfg.Link)
		s.send(ctx, tenantID, "whatsapp", phone, patientNumber, func() error {
			return s.whatsapp.Send(ctx, phone, body)
		})
	}
}

// send runs one channel and records the outcome. Errors are logged and audited,
// never returned — the caller is fire-and-forget by design.
func (s *Service) send(ctx context.Context, tenantID, channel, to, patientNumber string, fn func() error) {
	err := fn()
	action := "message.welcome_new_patient.sent"
	meta := map[string]any{"channel": channel, "to": to}
	if err != nil {
		action = "message.welcome_new_patient.failed"
		meta["error"] = err.Error()
		log.Printf("welcome %s to %s failed: %v", channel, to, err)
	}
	audit.LogRaw(ctx, s.repo.Tenants().Registry(), "", "", audit.Event{
		ActorKind: audit.ActorSystem,
		TenantID:  tenantID,
		Action:    action,
		Resource:  patientNumber,
		Metadata:  meta,
	})
}

// messageConfig reads one message type out of the clinic's communication settings.
// A clinic that has never saved settings simply has nothing enabled.
func (s *Service) messageConfig(ctx context.Context, tenantID, clinicID, key string) (channelConfig, bool) {
	if s.settings == nil {
		return channelConfig{}, false
	}
	pool, err := s.repo.Pool(ctx, tenantID)
	if err != nil {
		return channelConfig{}, false
	}
	raw, err := s.settings.GetJSONSettings(ctx, pool, "communication_settings", clinicID)
	if err != nil || len(raw) == 0 {
		return channelConfig{}, false
	}
	var parsed commsSettings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		log.Printf("communication_settings for clinic %s is unreadable: %v", clinicID, err)
		return channelConfig{}, false
	}
	cfg, ok := parsed.Messages[key]
	return cfg, ok
}

func (s *Service) clinicName(ctx context.Context, tenantID, clinicID string) string {
	pool, err := s.repo.Pool(ctx, tenantID)
	if err != nil {
		return "your clinic"
	}
	var name string
	if err := pool.QueryRow(ctx,
		`SELECT name FROM clinics WHERE id = $1::uuid`, clinicID,
	).Scan(&name); err != nil || strings.TrimSpace(name) == "" {
		return "your clinic"
	}
	return name
}

func primaryPhone(phones []model.PatientPhone) string {
	for _, p := range phones {
		if n := strings.TrimSpace(p.Number); n != "" {
			return strings.TrimSpace(p.CountryCode) + n
		}
	}
	return ""
}

// withLink appends the clinic's attached link, which is what the Attach Link
// modal on the Message Configuration screen stores.
func withLink(body, link string) string {
	if strings.TrimSpace(link) == "" {
		return body
	}
	return body + "\n\n" + link
}
