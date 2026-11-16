package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	thingscloud "github.com/arthursoares/things-cloud-sdk"
	"github.com/mark3labs/mcp-go/server"
)

func TestNewSingleUser(t *testing.T) {
	if u, err := newSingleUser("", ""); u != nil || err != nil {
		t.Fatalf("neither set: got %v, %v; want multi-user mode", u, err)
	}
	for _, c := range []struct{ email, password string }{
		{"owner@example.com", ""},
		{"", "testpass"},
		{"  ", "testpass"},
	} {
		if _, err := newSingleUser(c.email, c.password); err == nil {
			t.Errorf("newSingleUser(%q, %q) took half an account", c.email, c.password)
		}
	}
	u, err := newSingleUser(" owner@example.com ", "testpass")
	if err != nil || u.email != "owner@example.com" || u.password != "testpass" {
		t.Fatalf("got %+v, %v", u, err)
	}
}

func TestListenAddr(t *testing.T) {
	for _, c := range []struct {
		host, port string
		single     bool
		want       string
	}{
		{"", "", false, ":8080"},
		{"", "9000", false, ":9000"},
		{"", "", true, "127.0.0.1:8080"},
		{"", "8081", true, "127.0.0.1:8081"},
		{"0.0.0.0", "8081", true, "0.0.0.0:8081"},
		{"::1", "8081", true, "[::1]:8081"},
	} {
		if got := listenAddr(c.host, c.port, c.single); got != c.want {
			t.Errorf("listenAddr(%q, %q, %v) = %q, want %q", c.host, c.port, c.single, got, c.want)
		}
	}
}

func TestSingleUserContextIgnoresAuthorization(t *testing.T) {
	um := NewUserManager()
	um.single = &singleUser{email: "owner@example.com", password: "testpass"}
	for _, auth := range []string{
		"",
		"Basic " + base64.StdEncoding.EncodeToString([]byte("other@example.com:otherpass")),
		"Bearer some-token",
	} {
		r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		if auth != "" {
			r.Header.Set("Authorization", auth)
		}
		info, _ := um.httpContextFunc(context.Background(), r).Value(userContextKey).(*UserInfo)
		if info == nil || info.Email != "owner@example.com" || info.Password != "testpass" || info.Token != "" {
			t.Errorf("Authorization %q: got %+v, want the single user", auth, info)
		}
	}
}

func TestMCPHandlerAsksForAuthorizationOnlyWhenMultiUser(t *testing.T) {
	reached := false
	stream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })

	um := NewUserManager()
	w := httptest.NewRecorder()
	mcpHandler(um, stream)(w, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Header().Get("WWW-Authenticate"), "resource_metadata=") || reached {
		t.Fatalf("multi-user without Authorization: %d %q, reached %v", w.Code, w.Header().Get("WWW-Authenticate"), reached)
	}

	um.single = &singleUser{email: "owner@example.com", password: "testpass"}
	w = httptest.NewRecorder()
	mcpHandler(um, stream)(w, httptest.NewRequest(http.MethodPost, "/mcp", nil))
	if !reached {
		t.Fatalf("single-user without Authorization: %d, the request didn't reach the MCP server", w.Code)
	}
}

// A tool call without credentials, as an MCP proxy in front of the server
// makes it, reads the configured account's tasks.
func TestSingleUserToolCallReadsTheAccount(t *testing.T) {
	task := makeTaskItem("KXJnBwmWiWyVqd8YKR8wvm", withTitle("Water the plants"))
	fc := newFakeCloud("owner@example.com", task)
	defer fc.Close()

	um := NewUserManager()
	um.policy = newAccessPolicy("owner@example.com", "true")
	um.single = &singleUser{email: "owner@example.com", password: "testpass"}
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

	mcpServer := server.NewMCPServer("test", "0", server.WithToolCapabilities(false))
	mcpServer.AddTools(defineTools(um)...)
	stream := server.NewStreamableHTTPServer(mcpServer,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
		server.WithHTTPContextFunc(um.httpContextFunc),
	)
	srv := httptest.NewServer(mcpHandler(um, stream))
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"things_find_tasks","arguments":{}}}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %s", resp.StatusCode, raw)
	}
	var rpc struct {
		Result struct {
			IsError bool `json:"isError"`
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &rpc); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	if rpc.Result.IsError || len(rpc.Result.Content) == 0 || !strings.Contains(rpc.Result.Content[0].Text, "Water the plants") {
		t.Fatalf("tool call didn't read the account: %s", raw)
	}
}
