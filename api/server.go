package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/mods"
	"github.com/zaigie/palworld-server-tool/internal/tool"
)

// palServerProcessNames lists every process image name that belongs to a
// PalWorld dedicated-server session.  taskkill /T kills the whole tree so
// child processes (the actual game engine binary) are also terminated.
var palServerProcessNames = []string{
	"PalServer.exe",
	"PalServer-Win64-Shipping.exe",
	"PalServer-Win64-Shipping-Cmd.exe",
}

// killPalServerProcesses force-kills every surviving PalServer process tree.
// Errors are logged but not returned — the caller should not fail because of
// a process that is already gone.
func killPalServerProcesses() {
	for _, name := range palServerProcessNames {
		out, err := exec.Command("taskkill", "/F", "/IM", name, "/T").CombinedOutput()
		if err != nil {
			// "not found" is expected when the process is already gone; log at
			// debug level so the log isn't noisy during normal stop/start cycles.
			logger.Infof("[killPalServer] taskkill %s: %s", name, string(out))
		} else {
			logger.Infof("[killPalServer] terminated %s", name)
		}
	}
}

type ServerInfo struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

type ServerMetrics struct {
	ServerFps        int     `json:"server_fps"`
	CurrentPlayerNum int     `json:"current_player_num"`
	ServerFrameTime  float64 `json:"server_frame_time"`
	MaxPlayerNum     int     `json:"max_player_num"`
	Uptime           int     `json:"uptime"`
	Days             int     `json:"days"`
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
	info, err := tool.Info()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// TODO: add system psutil info
	c.JSON(http.StatusOK, &ServerInfo{info["version"], info["name"]})
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
	metrics, err := tool.Metrics()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, &ServerMetrics{
		ServerFps:        metrics["server_fps"].(int),
		CurrentPlayerNum: metrics["current_player_num"].(int),
		ServerFrameTime:  metrics["server_frame_time"].(float64),
		MaxPlayerNum:     metrics["max_player_num"].(int),
		Uptime:           metrics["uptime"].(int),
		Days:             metrics["days"].(int),
	})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	for _, dir := range candidates {
		if !palDefenderInstalled {
			// PalDefender ships as d3d9.dll + PalDefender.dll (or winhttp.dll) in the binary dir
			for _, fn := range []string{"PalDefender.dll", "d3d9.dll"} {
				if fileExists(filepath.Join(dir, fn)) {
					palDefenderInstalled = true
					break
				}
			}
		}
		if !ue4ssInstalled {
			// UE4SS ships as UE4SS.dll or dwmapi.dll (proxy) + UE4SS-settings.ini
			for _, fn := range []string{"UE4SS.dll", "UE4SS-settings.ini", "ue4ss.dll"} {
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
	//    (user set save.path = …\Pal\Binaries\Win64)
	add(cleanSave)

	// 2. Walk up the tree; at each ancestor check for a Binaries\Win64 subtree.
	//    This covers:
	//      …\Pal\Saved\SaveGames  -> up to …\Pal  -> …\Pal\Binaries\Win64
	//      …\Pal\Saved            -> up to …\Pal  -> …\Pal\Binaries\Win64
	cur := cleanSave
	for i := 0; i < 8; i++ {
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		add(filepath.Join(parent, "Binaries", "Win64"))
		add(filepath.Join(parent, "Pal", "Binaries", "Win64"))
		cur = parent
	}

	return dirs
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// startServer launches PalServer.exe derived from the configured save path.
//
//	@Summary		Start Server
//	@Description	Launch PalServer.exe (derived from save.path config)
//	@Tags			Server
//	@Produce		json
//	@Success		200	{object}	SuccessResponse
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/server/start [post]
func startServer(c *gin.Context) {
	cfg := config.Current()
	exePath := findServerExe(cfg.Save.Path)
	if exePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "PalServer.exe not found; check save.path configuration"})
		return
	}
	killPalServerProcesses()
	if err := launchDetached(exePath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// If a pending world-settings patch exists (written by the wizard), the
	// server will overwrite PalWorldSettings.ini on its first launch with
	// default values.  We watch the INI file in a background goroutine: as
	// soon as the server writes it (mtime changes and size > 100 bytes) we
	// re-merge our patch on top and clear the pending entry from the DB.
	store := config.CurrentStore()
	patchData := store.GetKV("pending_world_settings_patch")
	iniPathRaw := store.GetKV("pending_world_settings_ini")
	if len(patchData) > 0 && len(iniPathRaw) > 0 {
		iniPath := string(iniPathRaw)
		go watchAndReapplyWorldSettings(iniPath, patchData)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "path": exePath})
}

// watchAndReapplyWorldSettings polls iniPath until the server rewrites it,
// then merges patchData on top and removes the pending patch from the DB.
// It gives up after 10 minutes (server should have written it well before then).
func watchAndReapplyWorldSettings(iniPath string, patchData []byte) {
	var patch map[string]string
	if err := json.Unmarshal(patchData, &patch); err != nil || len(patch) == 0 {
		return
	}

	// Record the baseline state of the file before the server starts.
	baselineMtime := time.Time{}
	baselineSize := int64(0)
	if info, err := os.Stat(iniPath); err == nil {
		baselineMtime = info.ModTime()
		baselineSize = info.Size()
	}

	deadline := time.Now().Add(10 * time.Minute)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if time.Now().After(deadline) {
			return
		}
		info, err := os.Stat(iniPath)
		if err != nil {
			continue
		}
		// The server has rewritten the file when mtime is newer than our
		// baseline and the file is non-trivially large (>200 bytes).
		serverWroteIt := info.ModTime().After(baselineMtime) && info.Size() > 200
		// Also re-apply if the file didn't exist before (size was 0) and now it does.
		if baselineSize == 0 && info.Size() > 200 {
			serverWroteIt = true
		}
		if !serverWroteIt {
			continue
		}

		// Wait an extra second so the server finishes flushing the file.
		time.Sleep(1 * time.Second)

		existing, readErr := os.ReadFile(iniPath)
		if readErr != nil {
			continue
		}
		merged := mergeWorldSettings(string(existing), patch)
		if writeErr := os.WriteFile(iniPath, []byte(merged), 0644); writeErr != nil {
			continue
		}

		// Clear the pending patch so we don't re-apply on subsequent starts.
		store := config.CurrentStore()
		store.DeleteKV("pending_world_settings_patch")
		store.DeleteKV("pending_world_settings_ini")
		return
	}
}

// findServerExe locates PalServer.exe by:
//  1. Walking UP from savePath (handles normal layout: SaveGames is inside the server tree)
//  2. Walking UP and then DOWN into steamcmd/steamapps/common/PalServer (handles the
//     layout produced by our installer where steamcmd sits next to the chosen dir)
func findServerExe(savePath string) string {
	if savePath == "" {
		return ""
	}
	candidates := []string{"PalServer.exe", "PalServer", "PalServer-Win64-Shipping.exe"}

	tryDir := func(dir string) string {
		for _, name := range candidates {
			if full := filepath.Join(dir, name); fileExists(full) {
				return full
			}
		}
		return ""
	}

	// Pass 1: walk up the tree (covers "save.path is inside the server dir" layout)
	cur := filepath.Clean(savePath)
	for i := 0; i < 10; i++ {
		if hit := tryDir(cur); hit != "" {
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
		if hit := tryDir(probe); hit != "" {
			return hit
		}
		// Also check a sibling "steamcmd" directory (wizard puts steamcmd next to installDir)
		sibling := filepath.Join(filepath.Dir(cur), "steamcmd", "steamapps", "common", "PalServer")
		if hit := tryDir(sibling); hit != "" {
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

// serverRootFromSavePath walks from savePath to find the dir containing PalServer.exe.
func serverRootFromSavePath(savePath string) string {
	exe := findServerExe(savePath)
	if exe == "" {
		return ""
	}
	return filepath.Dir(exe)
}
