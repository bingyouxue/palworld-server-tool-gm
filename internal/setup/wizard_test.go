package setup

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestParsePublicBuildID(t *testing.T) {
	appInfo := `
"2394010"
{
    "depots"
    {
        "branches"
        {
            "public"
            {
                "buildid"        "17775555"
                "timeupdated"    "1740000000"
            }
        }
    }
}`
	if got := parsePublicBuildID(appInfo); got != "17775555" {
		t.Fatalf("parsePublicBuildID() = %q, want %q", got, "17775555")
	}
}

func TestParsePublicBuildIDDoesNotUseUnrelatedBuild(t *testing.T) {
	appInfo := `"buildid" "111" "branches" { "beta" { "buildid" "222" } }`
	if got := parsePublicBuildID(appInfo); got != "" {
		t.Fatalf("parsePublicBuildID() = %q, want empty", got)
	}
}

func TestNormalizePalServerInstallDirFromShippingBinaryDirectory(t *testing.T) {
	input := filepath.Join("E:\\PstServer", "Pal", "Binaries", "Win64")
	want := filepath.Clean("E:\\PstServer")
	if runtime.GOOS != "windows" {
		input = filepath.Join("/srv/PstServer", "Pal", "Binaries", "Linux")
		want = "/srv/PstServer"
	}
	if got := normalizePalServerInstallDir(input); got != want {
		t.Fatalf("normalizePalServerInstallDir(%q) = %q, want %q", input, got, want)
	}
}

func TestPalServerAppUpdateArgsPlacesInstallDirBeforeLogin(t *testing.T) {
	installDir := filepath.Join(t.TempDir(), "PalServer")
	args := palServerAppUpdateArgs(installDir)
	want := []string{
		"+@sSteamCmdForcePlatformType", steamPlatformType(),
		"+force_install_dir", installDir,
		"+login", "anonymous",
		"+app_update", "2394010", "validate",
		"+quit",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("palServerAppUpdateArgs() = %#v, want %#v", args, want)
	}
}

func TestDefaultSteamCMDDirIsOutsideServerRoot(t *testing.T) {
	serverRoot := filepath.Join(t.TempDir(), "PstServer")
	binaryDir := filepath.Join(serverRoot, "Pal", "Binaries", "Win64")
	want := filepath.Join(filepath.Dir(serverRoot), "steamcmd")
	if got := defaultSteamCMDDir(binaryDir); got != want {
		t.Fatalf("defaultSteamCMDDir(%q) = %q, want %q", binaryDir, got, want)
	}
}

func TestReadInstalledBuildID(t *testing.T) {
	root := t.TempDir()
	manifestDir := filepath.Join(root, "steamapps")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `"AppState" { "appid" "2394010" "buildid" "17775555" }`
	if err := os.WriteFile(filepath.Join(manifestDir, "appmanifest_2394010.acf"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	installDir := filepath.Join(root, "common", "PalServer")
	if got := readInstalledBuildID(installDir, filepath.Join(root, "steamcmd")); got != "17775555" {
		t.Fatalf("readInstalledBuildID() = %q, want %q", got, "17775555")
	}
}
