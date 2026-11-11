package main

import (
	"net/url"
	"testing"
)

func TestAccessPolicyAllowlist(t *testing.T) {
	open := newAccessPolicy("", "")
	if err := open.checkEmail("anyone@example.com"); err != nil {
		t.Fatalf("empty allowlist should allow everyone, got %v", err)
	}
	p := newAccessPolicy(" Owner@Example.com , second@example.com,", "")
	for _, ok := range []string{"owner@example.com", "OWNER@example.com ", "second@example.com"} {
		if err := p.checkEmail(ok); err != nil {
			t.Errorf("checkEmail(%q) = %v, want allowed", ok, err)
		}
	}
	if err := p.checkEmail("intruder@example.com"); err == nil {
		t.Error("checkEmail(intruder) allowed, want refused")
	}
}

func TestGetOrCreateUserRefusesBeforeLogin(t *testing.T) {
	um := NewUserManager()
	um.policy = newAccessPolicy("owner@example.com", "")
	called := false
	um.newUser = func(email, password string, proxyURL *url.URL) (*ThingsMCP, error) {
		called = true
		return &ThingsMCP{}, nil
	}
	if _, err := um.GetOrCreateUser("intruder@example.com", "pw"); err == nil {
		t.Fatal("expected refusal for account outside allowlist")
	}
	if called {
		t.Fatal("credentials for a refused account must not reach Things Cloud")
	}
	if _, err := um.GetOrCreateUser("owner@example.com", "pw"); err != nil {
		t.Fatalf("allowed account refused: %v", err)
	}
}

func TestReadOnlyRegistersOnlyReadTools(t *testing.T) {
	um := NewUserManager()
	um.policy = newAccessPolicy("", "true")
	tools := defineTools(um)
	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}
	for _, tool := range tools {
		hint := tool.Tool.Annotations.ReadOnlyHint
		if hint == nil || !*hint {
			t.Errorf("READ_ONLY registered write tool %s", tool.Tool.Name)
		}
	}
	all := defineTools(NewUserManager())
	if len(all) <= len(tools) {
		t.Fatalf("READ_ONLY did not remove any tools (%d of %d)", len(tools), len(all))
	}
}

func TestReadOnlyRefusesWrites(t *testing.T) {
	tm := &ThingsMCP{readOnly: true}
	if err := tm.writeAndSync(); err != errReadOnly {
		t.Fatalf("writeAndSync in read-only mode = %v, want errReadOnly", err)
	}
}
