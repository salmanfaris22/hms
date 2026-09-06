package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/salman/hms-backend/internal/modules/patient/model"
	"github.com/salman/hms-backend/pkg/messaging"
)

// The settings blob is written by the frontend; parsing must match its shape.
func TestParsesFrontendSettingsBlob(t *testing.T) {
	raw := []byte(`{"messages":{"welcome_new_patient":{"sms":true,"whatsapp":false,"email":true,"autoSend":true,"link":"https://x.test/a"}}}`)
	var parsed commsSettings
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	cfg, ok := parsed.Messages[messaging.KeyWelcomeNewPatient]
	if !ok {
		t.Fatal("welcome_new_patient not found")
	}
	if !cfg.SMS || cfg.WhatsApp || !cfg.Email || !cfg.AutoSend || cfg.Link != "https://x.test/a" {
		t.Fatalf("wrong config: %+v", cfg)
	}
}

func TestEmptyAndMissingSettingsAreInert(t *testing.T) {
	for _, raw := range []string{`{}`, `{"messages":{}}`} {
		var parsed commsSettings
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
		if _, ok := parsed.Messages[messaging.KeyWelcomeNewPatient]; ok {
			t.Fatalf("%s should yield no config", raw)
		}
	}
}

func TestPrimaryPhonePicksFirstNonEmpty(t *testing.T) {
	got := primaryPhone([]model.PatientPhone{
		{CountryCode: "+91", Number: "  "},
		{CountryCode: "+91", Number: " 9876543210 "},
	})
	if got != "+919876543210" {
		t.Fatalf("got %q", got)
	}
	if primaryPhone(nil) != "" {
		t.Fatal("nil phones should give empty")
	}
}

func TestRenderSubstitutesAndKeepsUnknowns(t *testing.T) {
	out := messaging.Render("Hi {{patientName}} ({{patientNumber}}) at {{clinicName}} {{nope}}", map[string]string{
		"patientName": "Asha", "patientNumber": "PA000123", "clinicName": "ClinicPro",
	})
	if out != "Hi Asha (PA000123) at ClinicPro {{nope}}" {
		t.Fatalf("got %q", out)
	}
}

func TestWelcomeTemplateHasAllThreeChannels(t *testing.T) {
	tmpl, ok := messaging.Get(messaging.KeyWelcomeNewPatient)
	if !ok {
		t.Fatal("welcome template missing")
	}
	for name, body := range map[string]string{
		"sms": tmpl.SMS, "whatsapp": tmpl.WhatsApp,
		"emailSubject": tmpl.EmailSubject, "emailBody": tmpl.EmailBody,
	} {
		if strings.TrimSpace(body) == "" {
			t.Fatalf("%s is empty", name)
		}
	}
	rendered := messaging.Render(tmpl.SMS, map[string]string{
		"clinicName": "ClinicPro", "patientName": "Asha", "patientNumber": "PA000123",
	})
	if strings.Contains(rendered, "{{") {
		t.Fatalf("unsubstituted placeholder: %q", rendered)
	}
}

func TestWithLinkAppendsOnlyWhenSet(t *testing.T) {
	if got := withLink("body", ""); got != "body" {
		t.Fatalf("got %q", got)
	}
	if got := withLink("body", "  "); got != "body" {
		t.Fatalf("blank link should not append, got %q", got)
	}
	if got := withLink("body", "https://x.test"); got != "body\n\nhttps://x.test" {
		t.Fatalf("got %q", got)
	}
}

func TestTextToHTMLEscapes(t *testing.T) {
	out := messaging.TextToHTML("a <b> & c")
	if strings.Contains(out, "<b>") || !strings.Contains(out, "&lt;b&gt;") || !strings.Contains(out, "&amp;") {
		t.Fatalf("not escaped: %q", out)
	}
}
