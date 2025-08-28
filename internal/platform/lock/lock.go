package lock

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
)

var file *os.File

func Acquire() error {
	path := filepath.Join("", "rsshub.lock")

	var err error
	file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	// Try to acquire exclusive lock
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return errors.New("another instance is already running")
	}
	return nil
}

func Release() error {
	if file == nil {
		return nil
	}
	defer file.Close()
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
