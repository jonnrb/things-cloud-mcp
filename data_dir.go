package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
)

// secureDataDir creates dataDir and makes it private to this process.
//
// The process can't chmod a directory it doesn't own, such as the root of a
// Kubernetes volume, which belongs to root and reaches the pod's group
// through fsGroup. Such a directory is used as it is unless anyone can write
// to it: the files the server keeps there are created private to it, but
// anyone who can write to the directory could replace them.
func secureDataDir(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return fmt.Errorf("create data directory %s: %w", dataDir, err)
	}
	chmodErr := os.Chmod(dataDir, 0700)
	if chmodErr == nil {
		return nil
	}
	if !errors.Is(chmodErr, fs.ErrPermission) {
		return fmt.Errorf("secure data directory %s: %w", dataDir, chmodErr)
	}
	info, err := os.Stat(dataDir)
	if err != nil {
		return fmt.Errorf("secure data directory %s: %w", dataDir, err)
	}
	warning, err := checkUnownedDataDir(info.Mode())
	if err != nil {
		return fmt.Errorf("data directory %s %w", dataDir, err)
	}
	if warning != "" {
		log.Printf("Warning: data directory %s %s", dataDir, warning)
	}
	return nil
}

// checkUnownedDataDir decides, from its mode, whether a data directory the
// process can't chmod is safe to use. Access for its group is the owner's
// choice, as with fsGroup. Read or search access for everyone gets a warning,
// since the files in it are private anyway. Write access for everyone is
// refused, even with the sticky bit, which would still let anyone plant a
// file before the server creates it.
func checkUnownedDataDir(mode fs.FileMode) (warning string, err error) {
	perm := mode.Perm()
	switch {
	case perm&0o002 != 0:
		return "", fmt.Errorf("isn't owned by this process, and its mode %#o lets anyone write to it", perm)
	case perm&0o005 != 0:
		return fmt.Sprintf("isn't owned by this process, so it keeps mode %#o, which lets anyone look inside; the files the server keeps there are still private to it", perm), nil
	default:
		return "", nil
	}
}

// createPrivateFile creates path if it's missing and makes it private to this
// process.
func createPrivateFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chmod(path, 0600)
}
