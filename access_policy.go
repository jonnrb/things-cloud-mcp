package main

import (
	"fmt"
	"os"
	"strings"
)

// accessPolicy holds deployment-level restrictions read from the environment.
//
//	ALLOWED_EMAILS: comma-separated Things Cloud account emails that may sign
//	in. When set, any other account is refused before its credentials are
//	sent to Things Cloud. Empty means no restriction (upstream behavior).
//
//	READ_ONLY: when "true" or "1", only tools annotated read-only are
//	registered, and every write to Things Cloud is refused.
type accessPolicy struct {
	allowed  map[string]bool
	readOnly bool
}

func loadAccessPolicy() accessPolicy {
	return newAccessPolicy(os.Getenv("ALLOWED_EMAILS"), os.Getenv("READ_ONLY"))
}

func newAccessPolicy(allowedEmails, readOnly string) accessPolicy {
	p := accessPolicy{}
	for _, e := range strings.Split(allowedEmails, ",") {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if p.allowed == nil {
			p.allowed = make(map[string]bool)
		}
		p.allowed[e] = true
	}
	switch strings.ToLower(strings.TrimSpace(readOnly)) {
	case "1", "true", "yes":
		p.readOnly = true
	}
	return p
}

// checkEmail returns an error when an allowlist is configured and email is not on it.
func (p accessPolicy) checkEmail(email string) error {
	if p.allowed == nil {
		return nil
	}
	if p.allowed[strings.ToLower(strings.TrimSpace(email))] {
		return nil
	}
	return fmt.Errorf("this server does not accept that Things Cloud account")
}

var errReadOnly = fmt.Errorf("this server is read-only; writes to Things Cloud are disabled")
