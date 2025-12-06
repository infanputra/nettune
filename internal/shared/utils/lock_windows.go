//go:build windows

package utils

import "sync"

type Mutex struct {
	mu   sync.Mutex
	name string
}

func NewMutex(name string) *Mutex {
	return &Mutex{name: name}
}

func (m *Mutex) Lock() {
	m.mu.Lock()
}

func (m *Mutex) Unlock() {
	m.mu.Unlock()
}

func (m *Mutex) TryLock() bool {
	return m.mu.TryLock()
}

type FileLock struct{}

func NewFileLock(path string) *FileLock {
	return &FileLock{}
}

func (fl *FileLock) Lock() error {
	return nil
}

func (fl *FileLock) TryLock() (bool, error) {
	return true, nil
}

func (fl *FileLock) Unlock() error {
	return nil
}
