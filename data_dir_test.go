package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecureDataDirCreatesPrivateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "oauth")
	if err := secureDataDir(dir); err != nil {
		t.Fatalf("secureDataDir: %v", err)
	}
	assertPerm(t, dir, 0o700)
}

func TestSecureDataDirTightensOwnedDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := secureDataDir(dir); err != nil {
		t.Fatalf("secureDataDir: %v", err)
	}
	assertPerm(t, dir, 0o700)
}

func TestCheckUnownedDataDir(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    fs.FileMode
		warn    bool
		refused bool
	}{
		{name: "private", mode: fs.ModeDir | 0o700},
		{name: "fsGroup on a 0700 volume root", mode: fs.ModeDir | fs.ModeSetgid | 0o770},
		{name: "group can read only", mode: fs.ModeDir | 0o750},
		{name: "fsGroup on a 0755 volume root", mode: fs.ModeDir | fs.ModeSetgid | 0o775, warn: true},
		{name: "anyone can search", mode: fs.ModeDir | 0o771, warn: true},
		{name: "anyone can write", mode: fs.ModeDir | 0o777, refused: true},
		{name: "anyone can write, sticky", mode: fs.ModeDir | fs.ModeSticky | 0o777, refused: true},
		{name: "anyone can write, nobody can read", mode: fs.ModeDir | 0o703, refused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			warning, err := checkUnownedDataDir(tc.mode)
			if refused := err != nil; refused != tc.refused {
				t.Fatalf("checkUnownedDataDir(%v) error = %v, want refused %v", tc.mode, err, tc.refused)
			}
			if warned := warning != ""; warned != tc.warn {
				t.Fatalf("checkUnownedDataDir(%v) warning = %q, want warning %v", tc.mode, warning, tc.warn)
			}
		})
	}
}

func TestCheckUnownedDataDirNamesTheMode(t *testing.T) {
	warning, _ := checkUnownedDataDir(fs.ModeDir | fs.ModeSetgid | 0o775)
	if !strings.Contains(warning, "0775") {
		t.Errorf("warning %q doesn't name mode 0775", warning)
	}
	_, err := checkUnownedDataDir(fs.ModeDir | 0o777)
	if err == nil || !strings.Contains(err.Error(), "0777") {
		t.Errorf("error %v doesn't name mode 0777", err)
	}
}

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
