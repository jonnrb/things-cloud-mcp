package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Every file the server keeps in the data directory is private on its own,
// including the WAL and shared-memory files SQLite creates, so they stay
// private in a directory others can look inside.
func TestOAuthServerFilesArePrivate(t *testing.T) {
	t.Setenv("CREDENTIALS_SECRET", "")
	t.Setenv("JWT_SECRET", "")
	dir := t.TempDir()
	o := NewOAuthServer(NewUserManager(), dir)
	t.Cleanup(func() { o.db.Close() })

	for _, name := range []string{"oauth.db", "oauth.db-wal", "oauth.db-shm", "credentials.key"} {
		assertPerm(t, filepath.Join(dir, name), 0o600)
	}
}

func TestCreatePrivateFileTightensExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth.db")
	if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := createPrivateFile(path); err != nil {
		t.Fatalf("createPrivateFile: %v", err)
	}
	assertPerm(t, path, 0o600)
	if got, err := os.ReadFile(path); err != nil || string(got) != "keep" {
		t.Fatalf("contents = %q, %v; want them kept", got, err)
	}
}

func assertPerm(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("%s has mode %#o, want %#o", filepath.Base(path), got, want)
	}
}
