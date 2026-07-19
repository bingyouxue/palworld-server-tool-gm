package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/mods"
	"github.com/zaigie/palworld-server-tool/internal/system"
	"github.com/zaigie/palworld-server-tool/internal/task"
	"github.com/zaigie/palworld-server-tool/internal/tool"
)

// palServerProcessNames returns process images used by the current platform.
// taskkill /T kills the whole Windows process tree, including engine children.
func palServerProcessNames() []string {
	if runtime.GOOS == "windows" {
		return []string{
			"PalServer.exe",
			"PalServer-Win64-Shipping.exe",
			"PalServer-Win64-Shipping-Cmd.exe",
		}
	}
	return []string{
		"PalServer.sh",
		"PalServer-Linux-Shipping",
	}
}

func isPalServerProcessRunning(name string) bool {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+name, "/NH", "/FO", "CSV").Output()
		return err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(name))
	}
	return exec.Command("pgrep", "-f", name).Run() == nil
}

// killPalServerProcesses force-kills every surviving PalServer process tree.
func killPalServerProcesses() {
	for _, name := range palServerProcessNames() {
		if !isPalServerProcessRunning(name) {
			continue
		}

		var out []byte
		var err error
		if runtime.GOOS == "windows" {
			out, err = exec.Command("taskkill", "/F", "/IM", name, "/T").CombinedOutput()
		} else {
			out, err = exec.Command("pkill", "-f", name).CombinedOutput()
		}
		if err != nil {
			logger.Warnf("[killPalServer] failed to terminate %s: %v; output=%s", name, err, strings.TrimSpace(string(out)))
			continue
		}
		logger.Infof("[killPalServer] terminated %s", name)
	}
}

func isPalServerRunning() bool {
	for _, name := range palServerProcessNames() {
		if isPalServerProcessRunning(name) {
			return true
		}
	}
	return false
}

type ServerInfo struct {
	Version             string `json:"version"`
	Platform            string `json:"platform"`
	Name                string `json:"name"`
	Running             bool   `json:"running"`
	ManagementAvailable bool   `json:"management_available"`
	ManagementError     string `json:"management_error,omitempty"`
}

type ServerMetrics struct {
	ServerFps           *int      `json:"server_fps"`
	CurrentPlayerNum    *int      `json:"current_player_num"`
	ServerFrameTime     *float64  `json:"server_frame_time"`
	MaxPlayerNum        *int      `json:"max_player_num"`
	Uptime              *int      `json:"uptime"`
	Days                *int      `json:"days"`
	CpuPercent          *float64  `json:"cpu_percent"`
	CpuTotalPercent     *float64  `json:"cpu_total_percent"`
	CpuPerCore          []float64 `json:"cpu_per_core"`
	MemoryBytes         uint64    `json:"memory_bytes"`
	MemoryTotalBytes    uint64    `json:"memory_total_bytes"`
	CpuCores            int       `json:"cpu_cores"`
	ProcessCount        int       `json:"process_count"`
	ProcessUptime       int64     `json:"process_uptime"`
	ManagementAvailable bool      `json:"management_available"`
	ManagementError     string    `json:"management_error,omitempty"`
}

type BroadcastRequest struct {
	Message string `json:"message"`
}

type ShutdownRequest struct {
	Seconds int    `json:"seconds"`
	Message string `json:"message"`
}

type ServerToolResponse struct {
	Version string `json:"version"`
	Latest  string `json:"latest"`
}

// getServerTool godoc
//
//	@Summary		Get PalWorld Server Tool
//	@Description	Get PalWorld Server Tool
//	@Tags			Server
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ServerToolResponse
//	@Router			/api/server/tool [get]
func getServerTool(c *gin.Context) {
	version, exists := c.Get("version")
	if !exists {
		version = "Unknown"
	}
	latest, err := tool.GetLatestTag()
	if err != nil {
		logger.Errorf("%v\n", err)
	}
	if latest == "" {
		latest, err = tool.GetLatestTagFromGitee()
		if err != nil {
			logger.Errorf("%v\n", err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"version": version, "latest": latest})
}

// getServer godoc
//
//	@Summary		Get Server Info
//	@Description	Get Server Info
//	@Tags			Server
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ServerInfo
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/server [get]
func getServer(c *gin.Context) {
	running := isPalServerRunning()
	info, err := tool.Info()
	if err != nil {
		if running {
			c.JSON(http.StatusOK, &ServerInfo{
				Running:             true,
				Platform:            runtime.GOOS,
				ManagementAvailable: false,
				ManagementError:     err.Error(),
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "running": false})
		return
	}
	c.JSON(http.StatusOK, &ServerInfo{
		Version:             info["version"],
		Platform:            runtime.GOOS,
		Name:                info["name"],
		Running:             true,
		ManagementAvailable: true,
	})
}

// getServerMetrics godoc
//
//	@Summary		Get Server Metrics
//	@Description	Get Server Metrics
//	@Tags			Server
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	ServerMetrics
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/server/metrics [get]
func getServerMetrics(c *gin.Context) {
	resources, resourceErr := system.GetPalServerResourceSnapshot()
	if resourceErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": resourceErr.Error()})
		return
	}

	response := &ServerMetrics{}
	if resources != nil {
		response.CpuPercent = resources.CPUPercent
		response.CpuTotalPercent = resources.CPUTotalPercent
		response.CpuPerCore = resources.CPUPerCore
		response.MemoryBytes = resources.MemoryBytes
		response.MemoryTotalBytes = resources.MemoryTotal
		response.CpuCores = resources.CPUCores
		response.ProcessCount = resources.ProcessCount
		response.ProcessUptime = resources.UptimeSeconds
	}

	metrics, err := tool.Metrics()
	if err != nil {
		if resources == nil {
			// 本机无进程且 REST 也不可达，服务器未运行
			c.JSON(http.StatusOK, gin.H{"running": false})
			return
		}
		response.ManagementError = err.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	serverFps := metrics["server_fps"].(int)
	currentPlayers := metrics["current_player_num"].(int)
	frameTime := metrics["server_frame_time"].(float64)
	maxPlayers := metrics["max_player_num"].(int)
	uptime := metrics["uptime"].(int)
	days := metrics["days"].(int)
	response.ServerFps = &serverFps
	response.CurrentPlayerNum = &currentPlayers
	response.ServerFrameTime = &frameTime
	response.MaxPlayerNum = &maxPlayers
	response.Uptime = &uptime
	response.Days = &days
	response.ManagementAvailable = true
	c.JSON(http.StatusOK, response)
}

// publishBroadcast godoc
//
//	@Summary		Publish Broadcast
//	@Description	Publish Broadcast
//	@Tags			Server
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			broadcast	body		BroadcastRequest	true	"Broadcast"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Router			/api/server/broadcast [post]
func publishBroadcast(c *gin.Context) {
	var req BroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateMessage(req.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := tool.Broadcast(req.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// shutdownServer godoc
//
//	@Summary		Shutdown Server
//	@Description	Shutdown Server
//	@Tags			Server
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			shutdown	body		ShutdownRequest	true	"Shutdown"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Router			/api/server/shutdown [post]
func shutdownServer(c *gin.Context) {
	var req ShutdownRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateMessage(req.Message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Seconds == 0 {
		req.Seconds = 60
	}
	if err := tool.Shutdown(req.Seconds, req.Message); err != nil {
		// A locally running PalServer must remain stoppable even when its REST
		// management endpoint is disabled, still starting, or misconfigured.
		if !isPalServerRunning() {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "running": false})
			return
		}
		logger.Warnf("[shutdownServer] graceful REST shutdown unavailable, forcing local process-tree stop: %v", err)
		killPalServerProcesses()
		time.Sleep(500 * time.Millisecond)
		if isPalServerRunning() {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":         "REST shutdown failed and the local PalServer process is still running",
				"rest_error":    err.Error(),
				"forced":        true,
				"still_running": true,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":    true,
			"forced":     true,
			"rest_error": err.Error(),
		})
		return
	}
	// After the RCON countdown, force-kill any surviving PalServer processes
	// so the status probe correctly shows the server as stopped.
	waitSecs := req.Seconds
	go func() {
		// Wait for the announced countdown plus a grace buffer.
		sleepSecs := waitSecs + 8
		if sleepSecs < 15 {
			sleepSecs = 15
		}
		timer := make(chan struct{})
		go func() {
			for i := 0; i < sleepSecs; i++ {
				time.Sleep(1 * time.Second)
			}
			close(timer)
		}()
		<-timer
		killPalServerProcesses()
	}()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func validateMessage(message string) error {
	if message == "" {
		return errors.New("message cannot be empty")
	}
	return nil
}

// getPluginStatus checks for PalDefender and UE4SS installation files on disk.
// It derives the server binary directory from the configured save path.
//
//	@Summary		Get Plugin Status
//	@Description	Check whether PalDefender and UE4SS are installed on disk
//	@Tags			Server
//	@Produce		json
//	@Success		200	{object}	map[string]bool
//	@Router			/api/server/plugins [get]
func getPluginStatus(c *gin.Context) {
	cfg := config.Current()
	savePath := cfg.Save.Path

	// Derive candidate server binary directories from the save path.
	// Common structures:
	//   <root>/Pal/Saved/SaveGames   -> binary at <root>/Pal/Binaries/Win64
	//   <root>/Pal/Saved             -> binary at <root>/Pal/Binaries/Win64
	//   <root>/Pal/Binaries/Win64    -> already the binary dir
	//   <root>/Pal                   -> binary at <root>/Pal/Binaries/Win64
	//   arbitrary path               -> walk up looking for Binaries/Win64

	candidates := deriveBinaryDirs(savePath)

	palDefenderInstalled := false
	ue4ssInstalled := false
	palDefenderFiles := []string{"PalDefender.dll", "d3d9.dll"}
	ue4ssFiles := []string{"UE4SS.dll", "UE4SS-settings.ini", "ue4ss.dll"}
	if runtime.GOOS != "windows" {
		palDefenderFiles = []string{"libPalDefender.so", "PalDefender.so"}
		ue4ssFiles = []string{"libUE4SS.so", "UE4SS.so"}
	}

	for _, dir := range candidates {
		if !palDefenderInstalled {
			// PalDefender ships as d3d9.dll + PalDefender.dll (or winhttp.dll) in the binary dir
			for _, fn := range palDefenderFiles {
				if fileExists(filepath.Join(dir, fn)) {
					palDefenderInstalled = true
					break
				}
			}
		}
		if !ue4ssInstalled {
			// UE4SS ships as UE4SS.dll or dwmapi.dll (proxy) + UE4SS-settings.ini
			for _, fn := range ue4ssFiles {
				if fileExists(filepath.Join(dir, fn)) {
					ue4ssInstalled = true
					break
				}
			}
		}
		if palDefenderInstalled && ue4ssInstalled {
			break
		}
	}

	// Read installed versions from the marker file in the first candidate dir
	// that actually exists (serverRootFromSavePath gives us the root).
	serverRoot := serverRootFromSavePath(savePath)
	marker := mods.ReadMarker(serverRoot)

	c.JSON(http.StatusOK, gin.H{
		"paldefender":         palDefenderInstalled,
		"ue4ss":               ue4ssInstalled,
		"paldefender_version": marker.PalDefender,
		"ue4ss_version":       marker.UE4SS,
		"search_dirs":         candidates,
	})
}

// deriveBinaryDirs returns a prioritised list of directories to search for
// plugin DLLs, given the configured save path.
func deriveBinaryDirs(savePath string) []string {
	if savePath == "" {
		return nil
	}

	var dirs []string
	seen := map[string]bool{}
	add := func(p string) {
		p = filepath.Clean(p)
		if !seen[p] {
			seen[p] = true
			dirs = append(dirs, p)
		}
	}

	cleanSave := filepath.Clean(savePath)

	// 1. The configured path itself may already be the binary directory
	add(cleanSave)

	binDir := "Win64"
	if runtime.GOOS != "windows" {
		binDir = "Linux"
	}

	// 2. Walk up the tree; at each ancestor check for a Binaries\Win64 subtree.
	cur := cleanSave
	for i := 0; i < 8; i++ {
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		add(filepath.Join(parent, "Binaries", binDir))
		add(filepath.Join(parent, "Pal", "Binaries", binDir))
		cur = parent
	}

	return dirs
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// startServer launches PalServer.exe derived from the configured save path.
// An optional JSON body field "mode" controls which binary is launched on Windows:
//   - "cmd"    → PalServer-Win64-Shipping-Cmd.exe  (console window visible)
//   - "silent" → PalServer-Win64-Shipping.exe       (no console window, default)
//
// On Linux the mode field is ignored and the normal PalServer.sh / PalServer binary is used.
//
//	@Summary		Start Server
//	@Description	Launch PalServer (derived from save.path config). mode: "cmd" or "silent" (Windows only)
//	@Tags			Server
//	@Produce		json
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/server/start [post]
func startServer(c *gin.Context) {
	// Parse optional mode field from request body
	var req struct {
		Mode string `json:"mode"`
	}
	// Best-effort bind; ignore errors (body may be empty)
	_ = c.ShouldBindJSON(&req)
	if req.Mode == "" || runtime.GOOS != "windows" {
		req.Mode = "silent"
	}

	if req.Mode != "silent" && req.Mode != "cmd" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported start mode %q", req.Mode)})
		return
	}

	cfg := config.Current()
	exePath := findServerExeByMode(cfg.Save.Path, req.Mode)
	if exePath == "" {
		exeName := "PalServer.sh"
		if runtime.GOOS == "windows" {
			exeName = "PalServer-Win64-Shipping.exe"
			if req.Mode == "cmd" {
				exeName = "PalServer-Win64-Shipping-Cmd.exe"
			}
		}
		errMessage := fmt.Sprintf("%s not found in server directory; configured save.path is %q", exeName, cfg.Save.Path)
		logger.Errorf("[startServer] %s", errMessage)
		c.JSON(http.StatusBadRequest, gin.H{"error": errMessage})
		return
	}

	logger.Infof("[startServer] requested mode=%s save.path=%q executable=%q", req.Mode, cfg.Save.Path, exePath)
	killPalServerProcesses()

	store := config.CurrentStore()
	patchData := store.GetKV("pending_world_settings_patch")
	iniPathRaw := store.GetKV("pending_world_settings_ini")
	var iniPath string
	var baselineMtime time.Time
	var baselineSize int64
	if len(patchData) > 0 && len(iniPathRaw) > 0 {
		iniPath = string(iniPathRaw)
		if err := applyWorldSettingsPatch(iniPath, patchData); err != nil {
			logger.Errorf("[startServer] apply world settings before launch: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "apply world settings before launch: " + err.Error()})
			return
		}
		if info, statErr := os.Stat(iniPath); statErr == nil {
			baselineMtime = info.ModTime()
			baselineSize = info.Size()
		}
		logger.Infof("[startServer] applied persisted world settings before launch ini=%q", iniPath)
	}

	logPath, err := launchServer(exePath, req.Mode)
	if err != nil {
		logger.Errorf("[startServer] launch failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "log_path": logPath})
		return
	}
	logger.Infof("[startServer] process started mode=%s executable=%q server_log=%q", req.Mode, exePath, logPath)

	// Keep watching briefly after launch because PalServer may rewrite its INI
	// during startup. The persisted patch is retained for every future launch.
	if iniPath != "" {
		go watchAndReapplyWorldSettings(iniPath, patchData, baselineMtime, baselineSize)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "path": exePath, "log_path": logPath})
}

func applyWorldSettingsPatch(iniPath string, patchData []byte) error {
	var patch map[string]string
	if err := json.Unmarshal(patchData, &patch); err != nil {
		return fmt.Errorf("decode persisted settings: %w", err)
	}
	if len(patch) == 0 {
		return errors.New("persisted world settings are empty")
	}

	existing, err := os.ReadFile(iniPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %q: %w", iniPath, err)
	}
	if len(existing) == 0 {
		existing = []byte("[/Script/Pal.PalGameWorldSettings]\nOptionSettings=()\n")
	}
	if err := os.MkdirAll(filepath.Dir(iniPath), 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	merged := mergeWorldSettings(string(existing), patch)
	if err := os.WriteFile(iniPath, []byte(merged), 0644); err != nil {
		return fmt.Errorf("write %q: %w", iniPath, err)
	}
	return nil
}

// watchAndReapplyWorldSettings restores persisted settings if PalServer
// rewrites its INI during startup. It gives up after 10 minutes.
func watchAndReapplyWorldSettings(iniPath string, patchData []byte, baselineMtime time.Time, baselineSize int64) {
	deadline := time.Now().Add(10 * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	var desired map[string]string
	if err := json.Unmarshal(patchData, &desired); err != nil || len(desired) == 0 {
		logger.Warnf("[startServer] cannot monitor world settings patch: %v", err)
		return
	}

	for range ticker.C {
		if time.Now().After(deadline) {
			return
		}
		info, err := os.Stat(iniPath)
		if err != nil {
			continue
		}
		data, readErr := os.ReadFile(iniPath)
		settingsChanged := readErr == nil && !worldSettingsContainPatch(string(data), desired)
		serverWroteIt := (info.ModTime().After(baselineMtime) && info.Size() > 200) || settingsChanged
		if baselineSize == 0 && info.Size() > 200 {
			serverWroteIt = true
		}
		if !serverWroteIt {
			continue
		}

		// Wait an extra second so the server finishes flushing the file.
		time.Sleep(1 * time.Second)

		if err := applyWorldSettingsPatch(iniPath, patchData); err != nil {
			logger.Warnf("[startServer] restore world settings after server rewrite: %v", err)
			continue
		}
		logger.Infof("[startServer] restored persisted world settings after server rewrite ini=%q", iniPath)
		return
	}
}

func worldSettingsContainPatch(ini string, desired map[string]string) bool {
	actual := parseIniToKV(ini)
	if len(actual) == 0 {
		return false
	}
	for key, value := range desired {
		if actual[key] != value {
			return false
		}
	}
	return true
}

// findServerExeByMode selects the appropriate PalServer binary based on mode.
// On Windows:
//   - "cmd"    → PalServer-Win64-Shipping-Cmd.exe  (console window)
//   - "silent" → PalServer-Win64-Shipping.exe       (no console window)
//
// On Linux the mode is ignored and findServerExe is called directly.
func findServerExeByMode(savePath, mode string) string {
	if runtime.GOOS != "windows" {
		return findServerExeLinux(savePath)
	}
	preferred := "PalServer-Win64-Shipping.exe"
	if mode == "cmd" {
		preferred = "PalServer-Win64-Shipping-Cmd.exe"
	}

	tryDir := func(dir string) string {
		full := filepath.Join(dir, preferred)
		if fileExists(full) {
			return full
		}
		return ""
	}

	// Also probe the standard Pal/Binaries/Win64 sub-tree from any ancestor.
	binSubDir := "Win64"
	tryDirDeep := func(root string) string {
		if hit := tryDir(root); hit != "" {
			return hit
		}
		// <root>/Pal/Binaries/Win64
		if hit := tryDir(filepath.Join(root, "Pal", "Binaries", binSubDir)); hit != "" {
			return hit
		}
		// <root>/Binaries/Win64  (in case root is already the Pal dir)
		if hit := tryDir(filepath.Join(root, "Binaries", binSubDir)); hit != "" {
			return hit
		}
		return ""
	}

	// Walk up from savePath
	cur := filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		if hit := tryDirDeep(cur); hit != "" {
			return hit
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	// Also check steamcmd layout
	cur = filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		for _, probe := range []string{
			filepath.Join(cur, "steamcmd", "steamapps", "common", "PalServer"),
			filepath.Join(filepath.Dir(cur), "steamcmd", "steamapps", "common", "PalServer"),
		} {
			if hit := tryDirDeep(probe); hit != "" {
				return hit
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// findServerExeLinux prefers the root PalServer.sh launcher because it sets
// the working directory and runtime library paths expected by the server.
func findServerExeLinux(savePath string) string {
	if savePath == "" {
		return ""
	}
	candidates := []string{"PalServer.sh", "PalServer", "PalServer-Linux-Shipping", "PalServer.exe"}
	findInDir := func(dir string) string {
		for _, name := range candidates {
			if full := filepath.Join(dir, name); fileExists(full) {
				return full
			}
		}
		for _, sub := range []string{filepath.Join(dir, "Pal", "Binaries", "Linux"), filepath.Join(dir, "Binaries", "Linux")} {
			for _, name := range candidates {
				if full := filepath.Join(sub, name); fileExists(full) {
					return full
				}
			}
		}
		return ""
	}
	cur := filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		if hit := findInDir(cur); hit != "" {
			return hit
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	cur = filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		for _, probe := range []string{filepath.Join(cur, "steamcmd", "steamapps", "common", "PalServer"), filepath.Join(filepath.Dir(cur), "steamcmd", "steamapps", "common", "PalServer")} {
			if hit := findInDir(probe); hit != "" {
				return hit
			}
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

// findServerExe locates the PalServer executable by:
//  1. Walking UP from savePath, checking each ancestor dir directly AND its
//     Pal/Binaries/Win64 (or Linux) sub-tree — this is the standard Steam layout
//     where the exe lives at <root>/Pal/Binaries/Win64/.
//  2. Walking UP and then DOWN into steamcmd/steamapps/common/PalServer (handles the
//     layout produced by our installer where steamcmd sits next to the chosen dir)
func findServerExe(savePath string) string {
	if savePath == "" {
		return ""
	}
	candidates := []string{"PalServer.exe", "PalServer.sh", "PalServer", "PalServer-Win64-Shipping.exe", "PalServer-Linux-Shipping"}

	tryDir := func(dir string) string {
		for _, name := range candidates {
			if full := filepath.Join(dir, name); fileExists(full) {
				return full
			}
		}
		return ""
	}

	// tryDirDeep checks dir itself plus the standard Pal/Binaries/{Win64,Linux} sub-paths.
	// This is required for the normal Steam layout where savePath is
	// <root>/Pal/Saved/SaveGames and the exe is at <root>/Pal/Binaries/Win64/.
	tryDirDeep := func(root string) string {
		if hit := tryDir(root); hit != "" {
			return hit
		}
		for _, sub := range []string{
			filepath.Join(root, "Pal", "Binaries", "Win64"),
			filepath.Join(root, "Pal", "Binaries", "Linux"),
			filepath.Join(root, "Binaries", "Win64"),
			filepath.Join(root, "Binaries", "Linux"),
		} {
			if hit := tryDir(sub); hit != "" {
				return hit
			}
		}
		return ""
	}

	// Pass 1: walk up the tree (covers "save.path is inside the server dir" layout)
	cur := filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		if hit := tryDirDeep(cur); hit != "" {
			return hit
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	// Pass 2: from each ancestor, also look inside steamcmd/steamapps/common/PalServer.
	// This covers the layout where the user chose E:\Foo as install dir and our wizard
	// put steamcmd in E:\steamcmd; the game ends up at E:\steamcmd\steamapps\common\PalServer.
	cur = filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		probe := filepath.Join(cur, "steamcmd", "steamapps", "common", "PalServer")
		if hit := tryDirDeep(probe); hit != "" {
			return hit
		}
		// Also check a sibling "steamcmd" directory (wizard puts steamcmd next to installDir)
		sibling := filepath.Join(filepath.Dir(cur), "steamcmd", "steamapps", "common", "PalServer")
		if hit := tryDirDeep(sibling); hit != "" {
			return hit
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}

	return ""
}

// serverRootFromSavePath walks from savePath to find the directory containing the PalServer executable.
func serverRootFromSavePath(savePath string) string {
	exe := findServerExe(savePath)
	if exe == "" {
		return ""
	}
	return filepath.Dir(exe)
}

// InitAutoRestart registers the server restart callback into the task package
// so that scheduled Shutdown RCON tasks automatically relaunch the server
// after it stops.  Call this once during startup, after config is loaded.
func InitAutoRestart() {
	task.SetRestartFunc(func(mode string) error {
		cfg := config.Current()
		exePath := findServerExeByMode(cfg.Save.Path, mode)
		if exePath == "" {
			return fmt.Errorf("PalServer executable not found for auto-restart; check save.path configuration")
		}
		killPalServerProcesses()
		if _, err := launchServer(exePath, mode); err != nil {
			return err
		}
		time.Sleep(3 * time.Second)
		if !isPalServerRunning() {
			return fmt.Errorf("PalServer process exited within 3 seconds after %s launch", mode)
		}
		return nil
	})
}
