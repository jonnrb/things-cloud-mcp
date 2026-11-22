package thingscloud

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"golang.org/x/time/rate"
)

// testHistoryKey is shaped like a real key; it isn't one.
const testHistoryKey = "11111111-2222-4333-8444-555555555555"

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(old) })
	return &buf
}

func TestRedactHistoryKeys(t *testing.T) {
	fp := HistoryFingerprint(testHistoryKey)
	for in, want := range map[string]string{
		"/version/1/history/" + testHistoryKey:                                      "/version/1/history/" + fp,
		"/version/1/history/" + testHistoryKey + "/items?start-index=0":             "/version/1/history/" + fp + "/items?start-index=0",
		"/version/1/history/" + testHistoryKey + "/commit?ancestor-index=12":        "/version/1/history/" + fp + "/commit?ancestor-index=12",
		`Get "https://cloud.example/version/1/history/` + testHistoryKey + `": EOF`: `Get "https://cloud.example/version/1/history/` + fp + `": EOF`,
		"GET /version/1/history/" + testHistoryKey + " HTTP/1.1":                    "GET /version/1/history/" + fp + " HTTP/1.1",
		"/version/1/account/a@example.com/own-history-keys/" + testHistoryKey:       "/version/1/account/a@example.com/own-history-keys/" + fp,
		"/version/1/account/a@example.com/own-history-keys":                         "/version/1/account/a@example.com/own-history-keys",
		"/version/1/history/" + fp + "/items":                                       "/version/1/history/" + fp + "/items",
	} {
		if got := RedactHistoryKeys(in); got != want {
			t.Errorf("RedactHistoryKeys(%q) = %q, want %q", in, got, want)
		}
	}
	if HistoryFingerprint(testHistoryKey) == HistoryFingerprint("11111111-2222-4333-8444-555555555556") {
		t.Error("two keys share a fingerprint")
	}
}

func TestErrorsLeaveOutTheHistoryKey(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	endpoint := ts.URL
	ts.Close() // every request now fails to connect

	_, err := newRetryTestClient(endpoint).History(testHistoryKey)
	if err == nil {
		t.Fatal("History succeeded against a closed server")
	}
	if strings.Contains(err.Error(), testHistoryKey) {
		t.Fatalf("error carries the history key: %v", err)
	}
	if !strings.Contains(err.Error(), HistoryFingerprint(testHistoryKey)) {
		t.Fatalf("error doesn't name the history by fingerprint: %v", err)
	}
}

func TestRetryLogLeavesOutTheHistoryKey(t *testing.T) {
	fastRetries(t)
	logs := captureLog(t)
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"latest-server-index": 1}`))
	}))
	defer ts.Close()

	if _, err := newRetryTestClient(ts.URL).History(testHistoryKey); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), "retrying") {
		t.Fatalf("no retry logged: %q", logs.String())
	}
	if strings.Contains(logs.String(), testHistoryKey) {
		t.Fatalf("retry log carries the history key: %q", logs.String())
	}
}

func TestDebugLeavesOutCredentials(t *testing.T) {
	logs := captureLog(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/version/1/account/") {
			w.Write([]byte(`{"status": "SYAccountStatusActive", "history-key": "` + testHistoryKey + `", "email": "user@example.com"}`))
			return
		}
		w.Write([]byte(`{"latest-server-index": 7}`))
	}))
	defer ts.Close()

	const password = "correct-horse-battery-staple"
	c := New(ts.URL, "user@example.com", password)
	c.rateLimiter = rate.NewLimiter(rate.Inf, 1)
	c.Debug = true
	v, err := c.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.History(v.HistoryKey); err != nil {
		t.Fatal(err)
	}

	out := logs.String()
	for _, secret := range []string{password, testHistoryKey} {
		if strings.Contains(out, secret) {
			t.Errorf("debug log carries %q:\n%s", secret, out)
		}
	}
	for _, want := range []string{"Authorization: [redacted]", HistoryFingerprint(testHistoryKey), `"latest-server-index": 7`} {
		if !strings.Contains(out, want) {
			t.Errorf("debug log lacks %q:\n%s", want, out)
		}
	}
}
