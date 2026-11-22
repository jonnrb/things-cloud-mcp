package thingscloud

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
)

// A history key is a bearer credential: Things Cloud serves a history's items,
// and takes commits to it, for anyone who puts the key in the request path,
// with no password. So the key stays out of everything this package logs or
// returns, and callers name a history by its HistoryFingerprint instead.

const fingerprintPrefix = "sha256:"

// HistoryFingerprint names a history key without revealing it: the first
// eight hex digits of the key's SHA-256.
func HistoryFingerprint(key string) string {
	sum := sha256.Sum256([]byte(key))
	return fingerprintPrefix + hex.EncodeToString(sum[:4])
}

// historyKeyInPath matches the key in /version/1/history/{key}/… and in
// /version/1/account/{email}/own-history-keys/{key}, in a path, a URL, or a
// message that quotes one.
var historyKeyInPath = regexp.MustCompile(`/(history|own-history-keys)/([^/?#\s"]+)`)

// RedactHistoryKeys replaces each history key in a Things Cloud path, URL or
// message with its HistoryFingerprint.
func RedactHistoryKeys(s string) string {
	return historyKeyInPath.ReplaceAllStringFunc(s, func(match string) string {
		m := historyKeyInPath.FindStringSubmatch(match)
		if strings.HasPrefix(m[2], fingerprintPrefix) {
			return match
		}
		return "/" + m[1] + "/" + HistoryFingerprint(m[2])
	})
}

// credentialHeader matches the header lines of a dump that carry a password
// or a session.
var credentialHeader = regexp.MustCompile(`(?mi)^(Authorization|Cookie|Set-Cookie):[^\r\n]*`)

// redactDump takes the credentials out of a Debug request or response dump.
func redactDump(dump []byte) string {
	return RedactHistoryKeys(credentialHeader.ReplaceAllString(string(dump), "${1}: [redacted]"))
}

// dumpsBody reports whether Debug may log a call's bodies. A history's own
// calls carry only Things data; the account and app-instance calls carry
// passwords, history keys and push tokens.
func dumpsBody(req *http.Request) bool {
	return strings.HasPrefix(req.URL.Path, "/version/1/history/")
}
