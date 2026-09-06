package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Sender dispatches transactional email. A SendGrid-backed implementation is
// used when an API key is configured; otherwise Log logs the payload so the
// dev loop can still see what would be sent.
type Sender interface {
	Send(ctx context.Context, to, subject, html string) error
}

func New(apiKey, fromEmail, fromName string) Sender {
	if apiKey == "" {
		return &logSender{from: fromEmail}
	}
	return &sendGrid{
		apiKey:   apiKey,
		from:     fromEmail,
		fromName: fromName,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

type logSender struct{ from string }

func (l *logSender) Send(_ context.Context, to, subject, html string) error {
	log.Printf("[email/log] (SENDGRID_API_KEY not set) to=%s from=%s subject=%q body_len=%d",
		to, l.from, subject, len(html))
	return nil
}

type sendGrid struct {
	apiKey   string
	from     string
	fromName string
	client   *http.Client
}

func (s *sendGrid) Send(ctx context.Context, to, subject, html string) error {
	body := map[string]any{
		"personalizations": []map[string]any{
			{
				"to":      []map[string]string{{"email": to}},
				"subject": subject,
			},
		},
		"from": map[string]string{
			"email": s.from,
			"name":  s.fromName,
		},
		"content": []map[string]string{
			{"type": "text/html", "value": html},
		},
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		"https://api.sendgrid.com/v3/mail/send",
		bytes.NewReader(buf),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("sendgrid %d: %s", res.StatusCode, string(b))
	}
	return nil
}

// WelcomeHTML renders the onboarding email sent to a brand-new hospital admin.
func WelcomeHTML(orgName, adminEmail, tempPassword, loginURL string) (subject, html string) {
	subject = fmt.Sprintf("Welcome to HMS — %s is ready", orgName)
	html = fmt.Sprintf(`<!doctype html>
<html><body style="font-family: Inter, Arial, sans-serif; background:#f4f4f7; padding:24px">
  <div style="max-width:560px; margin:0 auto; background:#fff; border-radius:16px; padding:28px; border:1px solid #eee">
    <h2 style="margin:0 0 8px; color:#1a1a2e">Welcome, %s</h2>
    <p style="color:#555; line-height:1.5">
      Your hospital organisation <strong>%s</strong> has been provisioned on the HMS platform.
      The database is live and your admin account is ready.
    </p>
    <div style="background:#f7f5ff; border:1px solid #e9e4ff; border-radius:12px; padding:16px; margin:16px 0">
      <div style="font-size:12px; color:#7a3dff; font-weight:700; text-transform:uppercase; letter-spacing:.5px">
        Your sign-in details
      </div>
      <div style="margin-top:8px; font-size:14px; color:#1a1a2e">
        <div><strong>Email:</strong> %s</div>
        <div><strong>Temporary password:</strong> <code>%s</code></div>
      </div>
      <p style="font-size:12px; color:#777; margin:12px 0 0">
        Please sign in and rotate this password immediately.
      </p>
    </div>
    <a href="%s" style="display:inline-block; background:#7a3dff; color:#fff; text-decoration:none;
       padding:12px 20px; border-radius:10px; font-weight:600">Sign in to HMS</a>
    <p style="color:#999; font-size:11px; margin-top:24px">
      If you did not request this account, ignore this email or contact your platform administrator.
    </p>
  </div>
</body></html>`, adminEmail, orgName, adminEmail, tempPassword, loginURL)
	return
}

// SubscriptionEndingHTML is the reminder template — wire a cron to trigger it
// before the subscription_end date.
func SubscriptionEndingHTML(orgName string, endsOn time.Time, loginURL string) (subject, html string) {
	subject = fmt.Sprintf("Action needed: %s subscription ends %s", orgName, endsOn.Format("Jan 2, 2006"))
	html = fmt.Sprintf(`<!doctype html>
<html><body style="font-family: Inter, Arial, sans-serif; background:#f4f4f7; padding:24px">
  <div style="max-width:560px; margin:0 auto; background:#fff; border-radius:16px; padding:28px; border:1px solid #eee">
    <h2 style="margin:0 0 8px; color:#1a1a2e">Your HMS subscription is ending</h2>
    <p style="color:#555; line-height:1.5">
      <strong>%s</strong> will stop accepting logins after <strong>%s</strong>.
      Renew now to avoid interruption to your clinic operations.
    </p>
    <a href="%s" style="display:inline-block; background:#7a3dff; color:#fff; text-decoration:none;
       padding:12px 20px; border-radius:10px; font-weight:600; margin-top:12px">Renew subscription</a>
  </div>
</body></html>`, orgName, endsOn.Format("Jan 2, 2006"), loginURL)
	return
}
