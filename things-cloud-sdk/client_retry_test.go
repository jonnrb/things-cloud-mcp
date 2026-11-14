package thingscloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func fastRetries(t *testing.T) {
	t.Helper()
	oldBase, oldMax := readRetryBaseDelay, readRetryMaxDelay
	readRetryBaseDelay, readRetryMaxDelay = time.Millisecond, 5*time.Millisecond
	t.Cleanup(func() { readRetryBaseDelay, readRetryMaxDelay = oldBase, oldMax })
}

func newRetryTestClient(url string) *Client {
	c := New(url, "user@example.com", "pw")
	c.rateLimiter = rate.NewLimiter(rate.Inf, 1)
	return c
}

func TestDoRetriesReadOn429(t *testing.T) {
	fastRetries(t)
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.Header().Set("Things-Response", "AbusePrevention")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	resp, err := newRetryTestClient(ts.URL).do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || atomic.LoadInt32(&calls) != 3 {
		t.Fatalf("status %d after %d calls, want 200 after 3", resp.StatusCode, calls)
	}
}

func TestDoGivesUpAfterMaxReadRetries(t *testing.T) {
	fastRetries(t)
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, "/x", nil)
	resp, err := newRetryTestClient(ts.URL).do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests || int(atomic.LoadInt32(&calls)) != maxReadRetries+1 {
		t.Fatalf("status %d after %d calls", resp.StatusCode, calls)
	}
}

func TestDoNeverRetriesWrites(t *testing.T) {
	fastRetries(t)
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "/x", nil)
	resp, err := newRetryTestClient(ts.URL).do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("POST was sent %d times, want 1", calls)
	}
}

func TestRetryDelayHonorsRetryAfterWithCap(t *testing.T) {
	fastRetries(t)
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"3600"}}}
	if d := retryDelay(resp, 0); d != readRetryMaxDelay {
		t.Fatalf("retryDelay = %s, want cap %s", d, readRetryMaxDelay)
	}
}
