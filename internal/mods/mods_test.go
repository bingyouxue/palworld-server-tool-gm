package mods

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWin64DirFromInstallationRoot(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "Pal", "Binaries", "Win64")
	if got := Win64Dir(root); got != want {
		t.Fatalf("Win64Dir(%q) = %q, want %q", root, got, want)
	}
}

func TestWin64DirDoesNotAppendToBinaryDirectory(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "Pal", "Binaries", "Win64")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if got := Win64Dir(binDir); got != binDir {
		t.Fatalf("Win64Dir(%q) = %q, want the binary directory unchanged", binDir, got)
	}
}

func TestWin64DirDetectsShippingExecutable(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "custom-bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(binDir, "PalServer-Win64-Shipping.exe")
	if err := os.WriteFile(exe, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got := Win64Dir(binDir); got != binDir {
		t.Fatalf("Win64Dir(%q) = %q, want executable directory unchanged", binDir, got)
	}
}
