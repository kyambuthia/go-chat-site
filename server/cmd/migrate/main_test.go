package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindProjectRoot_FindsNearestGoMod(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(tmp, "x", "y")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("findProjectRoot failed: %v", err)
	}
	if root != tmp {
		t.Fatalf("root = %q, want %q", root, tmp)
	}
}
