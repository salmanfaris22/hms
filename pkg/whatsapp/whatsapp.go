package whatsapp

import (
	"context"
	"log"
)

// Sender dispatches WhatsApp messages. No provider is integrated yet: until one
// is configured, New returns a logSender so the rest of the pipeline — settings
// gating, templating, auditing — runs and is observable in the dev loop.
//
// Adding a real provider (Meta WhatsApp Business API, or an aggregator) means implementing this interface
// and returning it from New when apiKey is set. Nothing else has to change.
type Sender interface {
	Send(ctx context.Context, to, body string) error
}

func New(apiKey, from string) Sender {
	if apiKey == "" {
		return &logSender{from: from}
	}
	// No provider implemented yet; fall back rather than silently dropping.
	log.Printf("[whatsapp] WHATSAPP_API_KEY is set but no provider is implemented — logging instead")
	return &logSender{from: from}
}

type logSender struct{ from string }

func (l *logSender) Send(_ context.Context, to, body string) error {
	log.Printf("[whatsapp/log] (no provider configured) to=%s from=%s body=%q", to, l.from, body)
	return nil
}
