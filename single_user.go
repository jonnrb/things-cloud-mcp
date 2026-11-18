package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

// singleUser is the one Things Cloud account a server in single-user mode acts
// as, for every request:
//
//	THINGS_EMAIL, THINGS_PASSWORD: the account. Set both for single-user mode,
//	or neither for the usual multi-user server.
//
// It's for a server behind something else that signs people in, such as an
// MCP proxy with its own OAuth. The server then needs no Authorization header,
// serves no OAuth endpoints, keeps nothing in DATA_DIR, and listens only on
// the loopback address unless HOST says otherwise, since whoever reaches it
// acts as the account.
type singleUser struct {
	email    string
	password string
}

func loadSingleUser() (*singleUser, error) {
	return newSingleUser(os.Getenv("THINGS_EMAIL"), os.Getenv("THINGS_PASSWORD"))
}

func newSingleUser(email, password string) (*singleUser, error) {
	email = strings.TrimSpace(email)
	switch {
	case email == "" && password == "":
		return nil, nil
	case email == "" || password == "":
		return nil, fmt.Errorf("single-user mode needs both THINGS_EMAIL and THINGS_PASSWORD")
	}
	return &singleUser{email: email, password: password}, nil
}

// listenAddr is where the server listens, from HOST and PORT. HOST defaults to
// every address, or to the loopback address in single-user mode.
func listenAddr(host, port string, single bool) string {
	if port == "" {
		port = "8080"
	}
	if host == "" && single {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// mcpHandler serves the MCP endpoint. Without an Authorization header it
// answers 401 with the protected resource's metadata, so a client can start
// OAuth, except in single-user mode, where no request needs one.
func mcpHandler(um *UserManager, stream http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if um.single == nil && r.Header.Get("Authorization") == "" {
			base := getBaseURL(r)
			w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+base+`/.well-known/oauth-protected-resource"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		stream.ServeHTTP(w, r)
	}
}
