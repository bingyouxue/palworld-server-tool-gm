// Package setup handles first-run wizard logic: reading existing server
// configs and installing a new server via SteamCMD.
package setup

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// ServerConfig holds the values parsed from an existing server installation.
type ServerConfig struct {
	ServerDir     string `json:"server_dir"`
	RconPort      int    `json:"rcon_port"`
	AdminPassword string `json:"admin_password"`
	RestPort      int    `json:"rest_port"`
	SavePath      string `json:"save_path"`
	// Raw key=value pairs from PalWorldSettings for further processing
	WorldSettings  map[string]string `json:"world_settings"`
	HasPalDefender bool              `json:"has_paldefender"`
	HasUE4SS       bool              `json:"has_ue4ss"`
}

// ValidateServerDir checks that a directory contains the PalServer executable.
func ValidateServerDir(dir string) error {
	candidates := []string{"PalServer.exe", "PalServer.sh", "PalServer-Linux-Shipping", "PalServer"}
	if runtime.GOOS != "windows" {
		candidates = []string{"PalServer.sh", "PalServer-Linux-Shipping", "PalServer", "PalServer.exe"}
	}
	for _, name := range candidates {
		if info, err := os.Stat(filepath.Join(dir, name)); err == nil && !info.IsDir() {
			return nil
		}
	}
	return fmt.Errorf("PalServer executable not found in %s (looked for: %v)", dir, candidates)
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
	platformConfigDir := "WindowsServer"
	if runtime.GOOS != "windows" {
		platformConfigDir = "LinuxServer"
	}
	settingsPath := filepath.Join(serverDir, "Pal", "Saved", "Config", platformConfigDir, "PalWorldSettings.ini")
	if data, err := os.ReadFile(settingsPath); err == nil {
		parseWorldSettings(string(data), &cfg)
	}

	// Derive save path
	cfg.SavePath = filepath.Join(serverDir, "Pal", "Saved", "SaveGames")

	// Check plugins in the platform-specific binary directory.
	binDir := filepath.Join(serverDir, "Pal", "Binaries", "Win64")
	palDefenderFiles := []string{"PalDefender.dll", "d3d9.dll"}
	ue4ssFiles := []string{"UE4SS.dll", "ue4ss.dll"}
	if runtime.GOOS != "windows" {
		binDir = filepath.Join(serverDir, "Pal", "Binaries", "Linux")
		palDefenderFiles = []string{"libPalDefender.so", "PalDefender.so"}
		ue4ssFiles = []string{"libUE4SS.so", "UE4SS.so"}
	}
	for _, filename := range palDefenderFiles {
		if _, err := os.Stat(filepath.Join(binDir, filename)); err == nil {
			cfg.HasPalDefender = true
			break
		}
	}
	for _, filename := range ue4ssFiles {
		if _, err := os.Stat(filepath.Join(binDir, filename)); err == nil {
			cfg.HasUE4SS = true
			break
		}
	}
	if !cfg.HasUE4SS {
		for _, filename := range ue4ssFiles {
			if _, err := os.Stat(filepath.Join(binDir, "ue4ss", filename)); err == nil {
				cfg.HasUE4SS = true
				break
			}
		}
	}

	return cfg, nil
}

var kvRe = regexp.MustCompile(`(\w+)=("(?:[^"\\]|\\.)*"|\([^)]*\)|[^,)]+)`)

func parseWorldSettings(ini string, cfg *ServerConfig) {
	start := strings.Index(ini, "OptionSettings=(")
	if start < 0 {
		return
	}
	innerStart := start + len("OptionSettings=(")
	depth, end := 1, -1
	inQuote := false
	for i := innerStart; i < len(ini); i++ {
		switch ini[i] {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote {
				depth--
				if depth == 0 {
					end = i
					i = len(ini)
				}
			}
		}
	}
	if end < innerStart {
		return
	}
	for _, kv := range kvRe.FindAllStringSubmatch(ini[innerStart:end], -1) {
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

	exeName := "steamcmd.exe"
	if runtime.GOOS != "windows" {
		exeName = "steamcmd.sh"
	}
	exePath := filepath.Join(destDir, exeName)
	if _, err := os.Stat(exePath); err == nil {
		progress("SteamCMD 已存在，跳过下载。")
		return exePath, nil
	}

	progress("正在下载 SteamCMD...")
	url := "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	if runtime.GOOS != "windows" {
		url = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("download steamcmd: %w", err)
	}
	defer resp.Body.Close()

	tmpFile := filepath.Join(destDir, "steamcmd_archive")
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return "", err
	}
	f.Close()
	progress("下载完成，正在解压...")

	if runtime.GOOS == "windows" {
		if err := unzip(tmpFile, destDir); err != nil {
			return "", fmt.Errorf("unzip steamcmd: %w", err)
		}
	} else {
		if err := untarGz(tmpFile, destDir); err != nil {
			return "", fmt.Errorf("untar steamcmd: %w", err)
		}
	}
	_ = os.Remove(tmpFile)
	progress("SteamCMD 准备就绪。")
	return exePath, nil
}

func untarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		fpath := filepath.Join(dest, filepath.Clean(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(fpath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
	return nil
}

// normalizePalServerInstallDir converts a binary/config subdirectory back to
// the root SteamCMD must receive via +force_install_dir.
func normalizePalServerInstallDir(dir string) string {
	if dir == "" {
		return ""
	}
	clean := filepath.Clean(dir)
	current := clean
	for i := 0; i < 10; i++ {
		if strings.EqualFold(filepath.Base(current), "Pal") {
			return filepath.Dir(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return clean
}

func defaultSteamCMDDir(installDir string) string {
	return filepath.Join(filepath.Dir(normalizePalServerInstallDir(installDir)), "steamcmd")
}

// InstallPalServer runs SteamCMD to install Palworld Dedicated Server.
// It streams output lines to progressFn.
func InstallPalServer(steamcmdPath, installDir string, progressFn func(string)) error {
	return runSteamCmd(steamcmdPath, installDir, progressFn, "install")
}

type PalServerVersionInfo struct {
	InstalledBuildID string `json:"installed_build_id"`
	LatestBuildID    string `json:"latest_build_id"`
	HasUpdate        bool   `json:"has_update"`
	ManifestFound    bool   `json:"manifest_found"`
}

// CheckPalServerVersion refreshes Steam app info and compares the installed
// manifest with the latest public branch without modifying server files.
func CheckPalServerVersion(installDir, steamCMDDir string, progressFn func(string)) (PalServerVersionInfo, error) {
	info := PalServerVersionInfo{}
	installDir = normalizePalServerInstallDir(installDir)
	if steamCMDDir == "" {
		steamCMDDir = defaultSteamCMDDir(installDir)
	}
	info.InstalledBuildID = readInstalledBuildID(installDir, steamCMDDir)
	info.ManifestFound = info.InstalledBuildID != ""

	steamcmdPath, err := DownloadSteamCMD(steamCMDDir, progressFn)
	if err != nil {
		return info, fmt.Errorf("prepare steamcmd: %w", err)
	}
	if err := initializeSteamCMD(steamcmdPath, progressFn); err != nil {
		return info, err
	}
	info.LatestBuildID, err = queryLatestPalServerBuildID(steamcmdPath, progressFn)
	if err != nil {
		return info, fmt.Errorf("获取 Steam public 分支最新版本失败: %w", err)
	}
	info.HasUpdate = info.InstalledBuildID == "" || info.InstalledBuildID != info.LatestBuildID
	return info, nil
}

// UpdatePalServer runs SteamCMD to update an existing Palworld Dedicated Server.
// Before and after app_update it refreshes Steam's app-info cache and verifies
// the installed manifest against the public branch build ID.
func UpdatePalServer(installDir, steamCMDDir string, progressFn func(string)) error {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	installDir = normalizePalServerInstallDir(installDir)
	if steamCMDDir == "" {
		steamCMDDir = defaultSteamCMDDir(installDir)
	}
	progress(fmt.Sprintf("服务端安装根目录：%s", installDir))
	progress("正在查找 SteamCMD...")
	steamcmdPath, err := DownloadSteamCMD(steamCMDDir, progressFn)
	if err != nil {
		return fmt.Errorf("prepare steamcmd: %w", err)
	}

	if err := initializeSteamCMD(steamcmdPath, progressFn); err != nil {
		return err
	}

	progress("正在刷新 Steam 应用信息并获取 public 分支最新版本...")
	latestBuildID, err := queryLatestPalServerBuildID(steamcmdPath, progressFn)
	if err != nil {
		return fmt.Errorf("获取服务端最新版本失败: %w", err)
	}
	installedBuildID := readInstalledBuildID(installDir, steamCMDDir)
	if installedBuildID == "" {
		progress(fmt.Sprintf("Steam public 分支最新 Build ID: %s；未找到本地版本清单，将执行完整校验更新。", latestBuildID))
	} else {
		progress(fmt.Sprintf("版本检查：本地 Build ID %s，Steam 最新 Build ID %s。", installedBuildID, latestBuildID))
		if installedBuildID == latestBuildID {
			progress("当前服务端已经是最新版本，仍将执行文件完整性校验。")
		}
	}

	progress("开始更新幻兽帕鲁服务端...")
	updateErr := runPalServerAppUpdate(steamcmdPath, installDir, progressFn)

	// The manifest is the source of truth. SteamCMD occasionally exits non-zero
	// after successfully updating itself or the app, so verify before deciding.
	updatedBuildID := readInstalledBuildID(installDir, steamCMDDir)
	if updatedBuildID == "" {
		if updateErr != nil {
			return fmt.Errorf("steamcmd: %w (更新后未找到 appmanifest_2394010.acf)", updateErr)
		}
		return fmt.Errorf("更新后未找到 appmanifest_2394010.acf，无法确认服务端版本")
	}
	if updatedBuildID != latestBuildID {
		if updateErr != nil {
			return fmt.Errorf("steamcmd: %w；更新后 Build ID 为 %s，Steam 最新为 %s", updateErr, updatedBuildID, latestBuildID)
		}
		return fmt.Errorf("更新未完成：本地 Build ID %s，Steam 最新 Build ID %s", updatedBuildID, latestBuildID)
	}

	progress(fmt.Sprintf("版本校验成功：已更新至 Steam public 分支最新 Build ID %s。", updatedBuildID))
	return nil
}

func initializeSteamCMD(steamcmdPath string, progressFn func(string)) error {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}
	progress("正在初始化 SteamCMD（首次运行会先下载自身运行环境，约 43 MB）...")
	if err := steamRun(steamcmdPath, progressFn, "+quit"); err != nil && !isSteamUpdateExit(err) {
		return fmt.Errorf("初始化 SteamCMD: %w", err)
	}
	return nil
}

func steamPlatformType() string {
	if runtime.GOOS == "windows" {
		return "windows"
	}
	return "linux"
}

func queryLatestPalServerBuildID(steamcmdPath string, progressFn func(string)) (string, error) {
	output, err := steamRunCapture(steamcmdPath, progressFn,
		"+@sSteamCmdForcePlatformType", steamPlatformType(),
		"+login", "anonymous",
		"+app_info_update", "1",
		"+app_info_print", "2394010",
		"+quit",
	)
	if err != nil && !isSteamUpdateExit(err) {
		return "", err
	}
	buildID := parsePublicBuildID(output)
	if buildID == "" {
		return "", errors.New("SteamCMD 返回的 app info 中没有 public 分支 buildid")
	}
	return buildID, nil
}

var (
	publicBranchBuildIDRe = regexp.MustCompile(`(?s)"branches"\s*\{.*?"public"\s*\{.*?"buildid"\s*"(\d+)"`)
	manifestBuildIDRe     = regexp.MustCompile(`(?i)"buildid"\s*"(\d+)"`)
)

func parsePublicBuildID(appInfo string) string {
	match := publicBranchBuildIDRe.FindStringSubmatch(appInfo)
	if len(match) == 2 {
		return match[1]
	}
	return ""
}

func readInstalledBuildID(installDir, steamCMDDir string) string {
	manifestName := "appmanifest_2394010.acf"
	var candidates []string
	addAncestorManifests := func(start string) {
		if start == "" {
			return
		}
		current := filepath.Clean(start)
		for i := 0; i < 8; i++ {
			candidates = append(candidates,
				filepath.Join(current, manifestName),
				filepath.Join(current, "steamapps", manifestName),
			)
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}
	addAncestorManifests(installDir)
	addAncestorManifests(steamCMDDir)

	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		match := manifestBuildIDRe.FindSubmatch(data)
		if len(match) == 2 {
			return string(match[1])
		}
	}
	return ""
}

func palServerAppUpdateArgs(installDir string) []string {
	return []string{
		"+@sSteamCmdForcePlatformType", steamPlatformType(),
		"+force_install_dir", normalizePalServerInstallDir(installDir),
		"+login", "anonymous",
		"+app_update", "2394010", "validate",
		"+quit",
	}
}

func runPalServerAppUpdate(steamcmdPath, installDir string, progressFn func(string)) error {
	return steamRun(steamcmdPath, progressFn, palServerAppUpdateArgs(installDir)...)
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

	steamPlatform := "windows"
	if runtime.GOOS != "windows" {
		steamPlatform = "linux"
	}

	// Pass 2: anonymous login to prime the platform/app manifest cache.
	// Without this a fresh SteamCMD reports "Missing configuration" (exit 8)
	// on the first real app_update because it hasn't downloaded the Steam
	// app-info depot configuration yet.
	progress("正在连接 Steam（匿名登录，用于获取应用配置）...")
	if err := steamRun(steamcmdPath, progressFn,
		"+@sSteamCmdForcePlatformType", steamPlatform,
		"+login", "anonymous",
		"+quit",
	); err != nil && !isSteamUpdateExit(err) {
		progress(fmt.Sprintf("[提示] 登录准备: %v（将继续尝试安装）", err))
	}

	// Pass 3: run the actual install / update.
	actionLabel := map[string]string{"install": "安装", "update": "更新"}[action]
	progress(fmt.Sprintf("开始%s到 %s ...", actionLabel, installDir))
	if err := steamRun(steamcmdPath, progressFn,
		"+@sSteamCmdForcePlatformType", steamPlatform,
		"+force_install_dir", installDir,
		"+login", "anonymous",
		"+app_update", "2394010", "validate",
		"+quit",
	); err != nil && !isSteamUpdateExit(err) {
		return fmt.Errorf("steamcmd: %w", err)
	}
	progress("幻兽帕鲁服务端操作完成！")
	return nil
}

// steamRun executes steamcmd with the given args, streaming every output line.
func steamRun(steamcmdPath string, progressFn func(string), args ...string) error {
	_, err := steamRunCapture(steamcmdPath, progressFn, args...)
	return err
}

// steamRunCapture streams SteamCMD output while also retaining it for parsing.
func steamRunCapture(steamcmdPath string, progressFn func(string), args ...string) (string, error) {
	progress := func(msg string) {
		if progressFn != nil {
			progressFn(msg)
		}
	}

	cmd := exec.Command(steamcmdPath, args...)
	cmd.Dir = filepath.Dir(steamcmdPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start steamcmd: %w", err)
	}

	var output strings.Builder
	var outputMu sync.Mutex
	drain := func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		// app_info_print may contain long lines in future SteamCMD versions.
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			progress(line)
			outputMu.Lock()
			output.WriteString(line)
			output.WriteByte('\n')
			outputMu.Unlock()
		}
	}

	var drains sync.WaitGroup
	drains.Add(2)
	go func() { defer drains.Done(); drain(stdout) }()
	go func() { defer drains.Done(); drain(stderr) }()
	waitErr := cmd.Wait()
	drains.Wait()
	return output.String(), waitErr
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
