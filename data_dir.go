package main

import (
	"os"
)

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
