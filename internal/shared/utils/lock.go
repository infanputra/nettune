package utils

import (
	"os"
	"sync"
	"syscall"
)

// Mutex is a simple mutex wrapper with naming for debugging
type Mutex struct {
	mu   sync.Mutex
	name string
}

// NewMutex creates a new named mutex
func NewMutex(name string) *Mutex {
	return &Mutex{name: name}
}

// Lock acquires the mutex
func (m *Mutex) Lock() {
	m.mu.Lock()
}

// Unlock releases the mutex
func (m *Mutex) Unlock() {
	m.mu.Unlock()
}

// TryLock attempts to acquire the mutex without blocking
func (m *Mutex) TryLock() bool {
	return m.mu.TryLock()
}

// FileLock provides file-based locking for cross-process synchronization
type FileLock struct {
	path string
	file *os.File
}

// NewFileLock creates a new file lock
func NewFileLock(path string) *FileLock {
	return &FileLock{path: path}
}

// Lock acquires the file lock
func (fl *FileLock) Lock() error {
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return err
	}

	fl.file = f
	return nil
}

// TryLock attempts to acquire the file lock without blocking
func (fl *FileLock) TryLock() (bool, error) {
	f, err := os.OpenFile(fl.path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return false, err
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return false, nil
		}
		return false, err
	}

	fl.file = f
	return true, nil
}

// Unlock releases the file lock
func (fl *FileLock) Unlock() error {
	if fl.file == nil {
		return nil
	}

	if err := syscall.Flock(int(fl.file.Fd()), syscall.LOCK_UN); err != nil {
		fl.file.Close()
		fl.file = nil
		return err
	}

	err := fl.file.Close()
	fl.file = nil
	return err
}
