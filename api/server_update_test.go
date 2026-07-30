package api

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestServerRootFromSavePathReturnsInstallRoot(t *testing.T) {
	root := t.TempDir()
	platformDir := "Win64"
	executable := "PalServer-Win64-Shipping.exe"
	if runtime.GOOS != "windows" {
		platformDir = "Linux"
		executable = "PalServer-Linux-Shipping"
	}
	binaryDir := filepath.Join(root, "Pal", "Binaries", platformDir)
	if err := os.MkdirAll(binaryDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binaryDir, executable), []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	savePath := filepath.Join(root, "Pal", "Saved", "SaveGames")
	if got := serverRootFromSavePath(savePath); got != root {
		t.Fatalf("serverRootFromSavePath(%q) = %q, want %q", savePath, got, root)
	}
}
