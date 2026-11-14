package main

import (
	"net/url"
	"testing"
	"time"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
)

func TestWarmUserBuildsStateFromTask7History(t *testing.T) {
	task := makeTaskItem("KXJnBwmWiWyVqd8YKR8wvm", withTitle("From a newer Things app"))
	task.Kind = thingscloud.ItemKindTask7
	fc := newFakeCloud("owner@example.com", task)
	defer fc.Close()

	um := NewUserManager()
	um.policy = newAccessPolicy("owner@example.com", "true")
	um.newUser = func(email, password string, _ *url.URL) (*ThingsMCP, error) {
		c := thingscloud.New(fc.server.URL, email, password, thingscloud.WithRequestInterval(0))
		if _, err := c.Verify(); err != nil {
			return nil, err
		}
		history, err := bestHistory(c)
		if err != nil {
			return nil, err
		}
		return &ThingsMCP{client: c, history: history}, nil
	}

	um.warmUser("owner@example.com", "testpass")

	deadline := time.Now().Add(5 * time.Second)
	for {
		um.mu.RLock()
		u := um.users["owner@example.com"]
		um.mu.RUnlock()
		if u != nil {
			if state := u.getState(); state != nil {
				got, ok := state.Tasks["KXJnBwmWiWyVqd8YKR8wvm"]
				if !ok || got.Title != "From a newer Things app" {
					t.Fatalf("warm state is missing the Task7 task: %#v", state.Tasks)
				}
				return
			}
		}
		if time.Now().After(deadline) {
			t.Fatal("warm-up didn't build the state")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestWarmUserSkipsAccountsOutsideTheAllowlist(t *testing.T) {
	um := NewUserManager()
	um.policy = newAccessPolicy("owner@example.com", "true")
	called := make(chan struct{}, 1)
	um.newUser = func(string, string, *url.URL) (*ThingsMCP, error) {
		called <- struct{}{}
		return nil, nil
	}
	um.warmUser("someone@example.com", "pw")
	select {
	case <-called:
		t.Fatal("warm-up signed in an account outside the allowlist")
	case <-time.After(200 * time.Millisecond):
	}
}
