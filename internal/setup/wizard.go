// Package setup handles first-run wizard logic: reading existing server
// configs and installing a new server via SteamCMD.
package setup

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ServerConfig holds the values parsed from an existing server installation.
type ServerConfig struct {
	ServerDir      string `json:"server_dir"`
	RconPort       int    `json:"rcon_port"`
	AdminPassword  string `json:"admin_password"`
	RestPort       int    `json:"rest_port"`
	SavePath       string `json:"save_path"`
	// Raw key=value pairs from PalWorldSettings for further processing
	WorldSettings  map[string]string `json:"world_settings"`
	HasPalDefender bool              `json:"has_paldefender"`
	HasUE4SS       bool              `json:"has_ue4ss"`
}

// ValidateServerDir checks that a directory contains PalServer.exe.
func ValidateServerDir(dir string) error {
	exe := filepath.Join(dir, "PalServer.exe")
	if _, err := os.Stat(exe); os.IsNotExist(err) {
		return fmt.Errorf("PalServer.exe not found in %s", dir)
	}
	return nil
}

// ParseServerConfig reads configuration from an existing installation.
func ParseServerConfig(serverDir string) (ServerConfig, error) {
	cfg := ServerConfig{
		ServerDir:     serverDir,
		RconPort:      25575,
		AdminPassword: "",
		RestPort:      8212,
		WorldSettings: map[string]string{},
	}

	// PalWorldSettings.ini
	settingsPath := filepath.Join(serverDir, "Pal", "Saved", "Config", "WindowsServer", "PalWorldSettings.ini")
	if data, err := os.ReadFile(settingsPath); err == nil {
		parseWorldSettings(string(data), &cfg)
	}

	// Derive save path
	cfg.SavePath = filepath.Join(serverDir, "Pal", "Saved", "SaveGames")

	// Check plugins
	win64 := filepath.Join(serverDir, "Pal", "Binaries", "Win64")
	if _, err := os.Stat(filepath.Join(win64, "PalDefender.dll")); err == nil {
		cfg.HasPalDefender = true
	}
	if _, err1 := os.Stat(filepath.Join(win64, "UE4SS.dll")); err1 == nil {
		cfg.HasUE4SS = true
	}
	if _, err2 := os.Stat(filepath.Join(win64, "ue4ss", "UE4SS.dll")); err2 == nil {
		cfg.HasUE4SS = true
	}

	return cfg, nil
}

var optionSettingsRe = regexp.MustCompile(`OptionSettings=\(([\s\S]*?)\)`)
var kvRe = regexp.MustCompile(`(\w+)=("(?:[^"\\]|\\.)*"|[^,)]+)`)

func parseWorldSettings(ini string, cfg *ServerConfig) {
	m := optionSettingsRe.FindStringSubmatch(ini)
	if len(m) < 2 {
		return
	}
	for _, kv := range kvRe.FindAllStringSubmatch(m[1], -1) {
		key := kv[1]
		val := strings.Trim(kv[2], `"`)
		cfg.WorldSettings[key] = val
		switch key {
		case "RCONPort":
			if p, err := strconv.Atoi(val); err == nil {
				cfg.RconPort = p
			}
		case "AdminPassword":
			cfg.AdminPassword = val
		case "RESTAPIPort":
			if p, err := strconv.Atoi(val); err == nil {
				cfg.RestPort = p
			}
		}
	}
}

// DownloadSteamCMD downloads and extracts steamcmd.zip to destDir.
// progressFn receives status messages.
func DownloadSteamCMD(destDir string, progressFn func(string)) (string, error) {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	exePath := filepath.Join(destDir, "steamcmd.exe")
	if _, err := os.Stat(exePath); err == nil {
		progress("SteamCMD 已存在，跳过下载。")
		return exePath, nil
	}

	progress("正在下载 SteamCMD...")
	const url = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download steamcmd: %w", err)
	}
	defer resp.Body.Close()

	tmpZip := filepath.Join(destDir, "steamcmd.zip")
	f, err := os.Create(tmpZip)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	progress("下载完成，正在解压...")

	if err := unzip(tmpZip, destDir); err != nil {
		return "", fmt.Errorf("unzip steamcmd: %w", err)
	}
	_ = os.Remove(tmpZip)
	progress("SteamCMD 准备就绪。")
	return exePath, nil
}

// InstallPalServer runs SteamCMD to install Palworld Dedicated Server.
// It streams output lines to progressFn.
func InstallPalServer(steamcmdPath, installDir string, progressFn func(string)) error {
	return runSteamCmd(steamcmdPath, installDir, progressFn, "install")
}

// UpdatePalServer runs SteamCMD to update an existing Palworld Dedicated Server.
// It finds steamcmd.exe by walking up from installDir/steamcmd, then falls back
// to downloading it fresh.
func UpdatePalServer(installDir string, progressFn func(string)) error {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}
	// Try to locate an existing steamcmd alongside or above the server dir
	steamDir := filepath.Join(filepath.Dir(installDir), "steamcmd")
	progress("正在查找 SteamCMD...")
	steamcmdPath, err := DownloadSteamCMD(steamDir, progressFn)
	if err != nil {
		return fmt.Errorf("prepare steamcmd: %w", err)
	}
	progress("开始更新幻兽帕鲁服务端...")
	return runSteamCmd(steamcmdPath, installDir, progress, "update")
}

func runSteamCmd(steamcmdPath, installDir string, progressFn func(string), action string) error {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}

	// Pass 1: let SteamCMD self-update.
	// On first run SteamCMD bootstraps itself (downloads ~43 MB runtime) and
	// exits with code 7. Any commands we pass are consumed by the bootstrapper,
	// so the actual app_update never runs. Running +quit alone lets it finish
	// cleanly; we ignore exit code 7 which is its normal self-update exit.
	progress("正在初始化 SteamCMD（首次运行会先下载自身运行环境，约 43 MB）...")
	if err := steamRun(steamcmdPath, progressFn, "+quit"); err != nil {
		// Exit 7 = self-update finished, completely normal
		if !isSteamUpdateExit(err) {
			progress(fmt.Sprintf("[提示] SteamCMD 初始化: %v", err))
		}
	}

	// Pass 2: anonymous login to prime the platform/app manifest cache.
	// Without this a fresh SteamCMD reports "Missing configuration" (exit 8)
	// on the first real app_update because it hasn't downloaded the Steam
	// app-info depot configuration yet.
	progress("正在连接 Steam（匿名登录，用于获取应用配置）...")
	if err := steamRun(steamcmdPath, progressFn,
		"+@sSteamCmdForcePlatformType", "windows",
		"+login", "anonymous",
		"+quit",
	); err != nil && !isSteamUpdateExit(err) {
		progress(fmt.Sprintf("[提示] 登录准备: %v（将继续尝试安装）", err))
	}

	// Pass 3: run the actual install / update.
	actionLabel := map[string]string{"install": "安装", "update": "更新"}[action]
	progress(fmt.Sprintf("开始%s到 %s ...", actionLabel, installDir))
	if err := steamRun(steamcmdPath, progressFn,
		"+@sSteamCmdForcePlatformType", "windows",
		"+login", "anonymous",
		"+force_install_dir", installDir,
		"+app_update", "2394010", "validate",
		"+quit",
	); err != nil && !isSteamUpdateExit(err) {
		return fmt.Errorf("steamcmd: %w", err)
	}
	progress("幻兽帕鲁服务端操作完成！")
	return nil
}

// steamRun executes steamcmd with the given args, streaming every output line
// to progressFn.  Both stdout and stderr are captured independently.
func steamRun(steamcmdPath string, progressFn func(string), args ...string) error {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	cmd := exec.Command(steamcmdPath, args...)
	cmd.Dir = filepath.Dir(steamcmdPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start steamcmd: %w", err)
	}

	// Drain stderr concurrently so it never blocks the process.
	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			progress(sc.Text())
		}
	}()

	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		progress(sc.Text())
	}

	return cmd.Wait()
}

// isSteamUpdateExit returns true for the exit codes SteamCMD uses when it has
// just updated itself (7) or when it completed an install but still needs a
// relaunch (1).  Both are non-fatal for our purposes.
func isSteamUpdateExit(err error) bool {
	if err == nil {
		return false
	}
	var ee *exec.ExitError
	if !strings.Contains(err.Error(), "exit status") {
		return false
	}
	_ = ee
	s := err.Error()
	return strings.HasSuffix(s, "exit status 7") ||
		strings.HasSuffix(s, "exit status 1")
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(dest, filepath.Clean(f.Name))
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(fpath, 0755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		out, err := os.Create(fpath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// SetupDoneKey is the bbolt key used to record wizard completion.
const SetupDoneKey = "setup_done"

// ReadSetupDone reads the setup_done flag from the given JSON store file.
// Returns false if not found or any error.
func ReadSetupDone(storePath string) bool {
	data, err := os.ReadFile(storePath)
	if err != nil {
		return false
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	v, _ := m[SetupDoneKey].(bool)
	return v
}
