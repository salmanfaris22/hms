// Package validator holds small re-usable validation helpers. Handlers build
// a list of issues and return them in one error so the client can render a
// per-field warning instead of whack-a-moling.
package validator

import (
	"net/mail"
	"regexp"
	"strings"
)

type Errors struct {
	Fields []string
}

func (e *Errors) Add(field string) { e.Fields = append(e.Fields, field) }

func (e *Errors) HasAny() bool { return len(e.Fields) > 0 }

func (e *Errors) Message() string {
	if len(e.Fields) == 0 {
		return ""
	}
	return "Missing or invalid: " + strings.Join(e.Fields, ", ")
}

func NotEmpty(v string) bool { return strings.TrimSpace(v) != "" }

func IsEmail(v string) bool {
	_, err := mail.ParseAddress(v)
	return err == nil
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)

func IsSlug(v string) bool { return slugRe.MatchString(v) }

func MinLen(v string, n int) bool { return len([]rune(v)) >= n }
