package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test", "nested", "dir")

	if err := EnsureDir(testDir); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	if !DirExists(testDir) {
		t.Error("Directory should exist after EnsureDir")
	}

	// Should not fail if already exists
	if err := EnsureDir(testDir); err != nil {
		t.Fatalf("EnsureDir failed on existing dir: %v", err)
	}
}

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// File doesn't exist yet
	if FileExists(tmpFile) {
		t.Error("File should not exist yet")
	}

	// Create file
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Now it should exist
	if !FileExists(tmpFile) {
		t.Error("File should exist after creation")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	if !DirExists(tmpDir) {
		t.Error("Temp dir should exist")
	}

	if DirExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("Nonexistent dir should not exist")
	}
}

func TestAtomicWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic.txt")
	content := []byte("atomic content")

	if err := AtomicWriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	// Verify content
	read, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(read) != string(content) {
		t.Errorf("File content = %q, want %q", string(read), string(content))
	}
}

func TestListFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	files := []string{"a.json", "b.json", "c.txt"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// List .json files
	jsonFiles, err := ListFiles(tmpDir, ".json")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(jsonFiles) != 2 {
		t.Errorf("Expected 2 json files, got %d", len(jsonFiles))
	}

	// List all files
	allFiles, err := ListFiles(tmpDir, "")
	if err != nil {
		t.Fatalf("ListFiles failed: %v", err)
	}

	if len(allFiles) != 3 {
		t.Errorf("Expected 3 files, got %d", len(allFiles))
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")
	content := []byte("copy me")

	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify destination content
	read, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}

	if string(read) != string(content) {
		t.Errorf("Dest content = %q, want %q", string(read), string(content))
	}
}
