// Package messaging holds the copy sent to patients, keyed by the same message
// type identifiers the Message Configuration screen uses
// (frontend: src/pages/settings-communication/CommunicationPage.tsx).
//
// Only welcome_new_patient has a trigger today; the rest are here so the later
// triggers (appointment confirmed, payment reminder, …) have somewhere to go.
package messaging

import "strings"

// Keys must match the frontend's MESSAGE_TYPES entries exactly — they are the
// keys inside the clinic's communication_settings blob.
const (
	KeyWelcomeNewPatient    = "welcome_new_patient"
	KeyAppointmentConfirmed = "appointment_confirmation"
	KeyAppointmentCancelled = "appointment_cancellation"
	KeyAppointmentReminder  = "appointment_reminder"
	KeyDocumentsReady       = "documents_ready"
	KeyPaymentReminder      = "payment_reminder"
	KeyBirthdayWishes       = "birthday_wishes"
	KeyFollowUp             = "follow_up"
	KeyReviewMessage        = "review_message"
)

// Template is the copy for one message type across the three channels.
type Template struct {
	SMS          string
	WhatsApp     string
	EmailSubject string
	EmailBody    string // plain text; converted to simple HTML on send
}

var templates = map[string]Template{
	KeyWelcomeNewPatient: {
		SMS: "Welcome to {{clinicName}}, {{patientName}}! Your patient ID is {{patientNumber}}. " +
			"Keep it handy for your visits.",
		WhatsApp: "Hi {{patientName}},\n\nWelcome to {{clinicName}}!\n\n" +
			"Your patient ID is {{patientNumber}}. Please keep it handy for appointments and reports.\n\n" +
			"We look forward to caring for you.",
		EmailSubject: "Welcome to {{clinicName}}",
		EmailBody: "Dear {{patientName}},\n\nWelcome to {{clinicName}}.\n\n" +
			"Your registration is complete and your patient ID is {{patientNumber}}. " +
			"Please quote it when booking appointments or collecting reports.\n\n" +
			"If you have any questions, just reply to this email.\n\n" +
			"Best regards,\n{{clinicName}}",
	},
}

// Get returns the template for a message type, and whether one is defined.
func Get(key string) (Template, bool) {
	t, ok := templates[key]
	return t, ok
}

// Render substitutes {{placeholder}} tokens. Unknown placeholders are left as-is
// so a typo is visible in the output rather than silently blanking the message.
func Render(s string, vars map[string]string) string {
	if s == "" || len(vars) == 0 {
		return s
	}
	pairs := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		pairs = append(pairs, "{{"+k+"}}", v)
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// TextToHTML wraps plain-text body copy for the HTML-only email sender.
func TextToHTML(s string) string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:Arial,sans-serif;font-size:14px;line-height:1.5;color:#111827">`)
	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			b.WriteString("<br/>")
			continue
		}
		b.WriteString("<p style=\"margin:0 0 8px\">")
		b.WriteString(htmlEscape(line))
		b.WriteString("</p>")
	}
	b.WriteString("</div>")
	return b.String()
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
