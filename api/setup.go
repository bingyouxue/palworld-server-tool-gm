package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/setup"
)

// setupDone is stored in config.db as a flag key in the config bucket.
// We piggyback on the existing bbolt store via a dedicated key.
const setupDoneConfigKey = "__setup_done__"

func isSetupDone() bool {
	return config.CurrentStore().GetFlag(setupDoneConfigKey)
}

func markSetupDone() {
	config.CurrentStore().SetFlag(setupDoneConfigKey, true)
}

// getSetupStatus godoc
//
//	@Summary		Get first-run wizard status
//	@Tags			Setup
//	@Produce		json
//	@Router			/api/setup/status [get]
func getSetupStatus(c *gin.Context) {
	done := isSetupDone()
	c.JSON(http.StatusOK, gin.H{"done": done})
}

// adoptServerRequest is the body for POST /api/setup/adopt
type adoptServerRequest struct {
	ServerDir string `json:"server_dir" binding:"required"`
}

// postSetupAdopt validates a server directory, reads its config and writes
// it back into PST settings.
//
//	@Summary		Adopt existing server installation
//	@Tags			Setup
//	@Accept			json
//	@Produce		json
//	@Router			/api/setup/adopt [post]
func postSetupAdopt(c *gin.Context) {
	var req adoptServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := setup.ValidateServerDir(req.ServerDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parsed, err := setup.ParseServerConfig(req.ServerDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Write parsed values into PST config
	store := config.CurrentStore()
	cfg := store.Config()

	cfg.Rcon.Address = fmt.Sprintf("127.0.0.1:%d", parsed.RconPort)
	cfg.Rcon.Password = parsed.AdminPassword
	cfg.Rest.Address = fmt.Sprintf("http://127.0.0.1:%d", parsed.RestPort)
	cfg.Rest.Password = parsed.AdminPassword
	cfg.Save.Path = parsed.SavePath

	if err := store.Update(cfg, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save config: " + err.Error()})
		return
	}

	markSetupDone()
	c.JSON(http.StatusOK, gin.H{"success": true, "parsed": parsed})
}

// installRequest is the body for POST /api/setup/install
type installRequest struct {
	InstallDir string `json:"install_dir" binding:"required"`
	SteamCMDir string `json:"steamcmd_dir"` // optional, defaults to installDir/steamcmd
}

// install progress channels keyed by a simple counter
var (
	installMu   sync.Mutex
	installChs  = map[int]chan string{}
	installIdx  int
)

// postSetupInstall triggers async download + install of Palworld dedicated server.
//
//	@Summary		Download SteamCMD and install Palworld Dedicated Server
//	@Tags			Setup
//	@Accept			json
//	@Produce		json
//	@Router			/api/setup/install [post]
func postSetupInstall(c *gin.Context) {
	var req installRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// SteamCMD lives in a sibling "steamcmd" folder next to the chosen install dir.
	// The game is installed into <installDir> directly via +force_install_dir.
	// of its working directory.
	installDirAbs, err := filepath.Abs(req.InstallDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid install_dir: " + err.Error()})
		return
	}

	steamDir := req.SteamCMDir
	if steamDir == "" {
			steamDir = filepath.Join(filepath.Dir(installDirAbs), "steamcmd")
	}

	installMu.Lock()
	installIdx++
	id := installIdx
	ch := make(chan string, 128)
	installChs[id] = ch
	installMu.Unlock()

	go func() {
		defer func() {
			installMu.Lock()
			delete(installChs, id)
			installMu.Unlock()
			close(ch)
		}()

		send := func(msg string) { ch <- msg }

		steamcmdPath, err := setup.DownloadSteamCMD(steamDir, send)
		if err != nil {
			ch <- "[错误] " + err.Error()
			return
		}
		if err := setup.InstallPalServer(steamcmdPath, installDirAbs, send); err != nil {
			ch <- "[错误] " + err.Error()
			return
		}

		// After a successful install, automatically write the PST config so that
		// save.path, RCON and REST are pre-populated.  The wizard's step 3 will
		// overwrite these again with the user-edited world settings, but having
		// them set here means that clicking "install mod" before step 3 also works.
		serverCfg, parseErr := setup.ParseServerConfig(installDirAbs)
		if parseErr == nil {
			store := config.CurrentStore()
			cfg := store.Config()
			cfg.Save.Path = serverCfg.SavePath
			cfg.Rcon.Address = fmt.Sprintf("127.0.0.1:%d", serverCfg.RconPort)
			cfg.Rcon.Password = serverCfg.AdminPassword
			cfg.Rest.Address = fmt.Sprintf("http://127.0.0.1:%d", serverCfg.RestPort)
			cfg.Rest.Password = serverCfg.AdminPassword
			_ = store.Update(cfg, "")
		}

		// Emit the actual server directory so the frontend can pass it back in
		// the world-settings PUT request.
		ch <- fmt.Sprintf("[server_dir] %s", installDirAbs)
		ch <- "[完成] install_done"
	}()

	c.JSON(http.StatusOK, gin.H{"install_id": id})
}

// getSetupInstallProgress streams install progress as SSE.
//
//	@Summary		SSE stream for install progress
//	@Tags			Setup
//	@Produce		text/event-stream
//	@Router			/api/setup/install/progress/{id} [get]
func getSetupInstallProgress(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	if _, err := fmt.Sscan(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	installMu.Lock()
	ch, ok := installChs[id]
	installMu.Unlock()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	if !ok {
		c.SSEvent("error", "install not found or already finished")
		c.Writer.Flush()
		return
	}

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, more := <-ch:
			if !more {
				c.SSEvent("done", "")
				c.Writer.Flush()
				return
			}
			c.SSEvent("log", msg)
			c.Writer.Flush()
		case <-ticker.C:
			c.SSEvent("ping", "")
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}

// postSetupComplete marks the wizard as done without doing anything else.
//
//	@Summary		Mark setup wizard as completed
//	@Tags			Setup
//	@Produce		json
//	@Router			/api/setup/complete [post]
func postSetupComplete(c *gin.Context) {
	markSetupDone()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// serverUpdateRequest is the body for POST /api/setup/server-update
type serverUpdateRequest struct {
	ServerDir  string `json:"server_dir"`  // optional, derived from save.path if empty
	SteamCMDir string `json:"steamcmd_dir"` // optional
}

// postSetupServerUpdate triggers an async steamcmd app_update for the server.
//
//	@Summary		Update Palworld Dedicated Server via SteamCMD
//	@Tags			Setup
//	@Accept			json
//	@Produce		json
//	@Router			/api/setup/server-update [post]
func postSetupServerUpdate(c *gin.Context) {
	var req serverUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Derive install dir from save.path if not explicitly given
	serverDir := req.ServerDir
	if serverDir == "" {
		cfg := config.Current()
		serverDir = serverRootFromSavePath(cfg.Save.Path)
	}
	if serverDir == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法确定服务器目录，请手动指定 server_dir"})
		return
	}

	installMu.Lock()
	installIdx++
	id := installIdx
	ch := make(chan string, 256)
	installChs[id] = ch
	installMu.Unlock()

	go func() {
		defer func() {
			installMu.Lock()
			delete(installChs, id)
			installMu.Unlock()
			close(ch)
		}()

		send := func(msg string) { ch <- msg }

		if err := setup.UpdatePalServer(serverDir, send); err != nil {
			ch <- "[错误] " + err.Error()
			return
		}
		ch <- "[完成] update_done"
	}()

	c.JSON(http.StatusOK, gin.H{"install_id": id})
}

// worldSettingsRequest is the body for PUT /api/setup/world-settings
type worldSettingsRequest struct {
	// ServerDir is used when save.path is not yet configured (install flow).
	// If empty, path is derived from the configured save.path.
	ServerDir string            `json:"server_dir"`
	Settings  map[string]string `json:"settings" binding:"required"`
}

// putSetupWorldSettings writes key=value pairs into PalWorldSettings.ini.
// It reads the existing file, replaces values inside the OptionSettings=(…)
// block, then writes it back. Creates the file with defaults if it doesn't exist.
//
//	@Summary		Write PalWorldSettings.ini from key-value map
//	@Tags			Setup
//	@Accept			json
//	@Produce		json
//	@Router			/api/setup/world-settings [put]
func putSetupWorldSettings(c *gin.Context) {
	var req worldSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Determine the ini file path
	iniPath := ""
	serverDirName := "WindowsServer"
	if runtime.GOOS != "windows" {
		serverDirName = "LinuxServer"
	}
	if req.ServerDir != "" {
		iniPath = filepath.Join(req.ServerDir, "Pal", "Saved", "Config", serverDirName, "PalWorldSettings.ini")
	} else {
		savePath := config.Current().Save.Path
		if savePath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "save.path not configured and server_dir not provided"})
			return
		}
		palDir := findPalDir(savePath)
		if palDir == "" {
			palDir = filepath.Clean(savePath)
		}
		iniPath = filepath.Join(palDir, "Saved", "Config", serverDirName, "PalWorldSettings.ini")
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(iniPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Read existing content or start with a skeleton
	existing := ""
	if data, err := os.ReadFile(iniPath); err == nil {
		existing = string(data)
	}
	if existing == "" {
		existing = "[/Script/Pal.PalGameWorldSettings]\nOptionSettings=()\n"
	}

	updated := mergeWorldSettings(existing, req.Settings)
	if err := os.WriteFile(iniPath, []byte(updated), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also write PST runtime config so RCON/REST/save.path take effect immediately.
	// Derive the server directory from server_dir or from the existing save.path.
	serverDir := req.ServerDir
	if serverDir == "" {
		serverDir = serverRootFromSavePath(config.Current().Save.Path)
	}
	if serverDir != "" {
		// Rebuild save path pointing at the SaveGames directory
		savePath := filepath.Join(serverDir, "Pal", "Saved", "SaveGames")

		rconPort := "25575"
		restPort := "8212"
		adminPwd := ""
		if v, ok := req.Settings["RCONPort"]; ok && v != "" {
			rconPort = v
		}
		if v, ok := req.Settings["RESTAPIPort"]; ok && v != "" {
			restPort = v
		}
		if v, ok := req.Settings["AdminPassword"]; ok {
			adminPwd = v
		}

		store := config.CurrentStore()
		cfg := store.Config()
		cfg.Save.Path = savePath
		cfg.Rcon.Address = fmt.Sprintf("127.0.0.1:%s", rconPort)
		cfg.Rcon.Password = adminPwd
		cfg.Rest.Address = fmt.Sprintf("http://127.0.0.1:%s", restPort)
		cfg.Rest.Password = adminPwd
		// Ignore error — best-effort; wizard will still succeed
		_ = store.Update(cfg, "")

		// Persist the desired world settings as a "pending patch" so that
		// startServer can re-apply them after the server's first launch
		// overwrites the INI with its own defaults.
		if patchData, jsonErr := json.Marshal(req.Settings); jsonErr == nil {
			_ = store.SetKV("pending_world_settings_patch", patchData)
			_ = store.SetKV("pending_world_settings_ini", []byte(iniPath))
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "path": iniPath})
}

// mergeWorldSettings merges kv pairs into an INI string's OptionSettings block.
func mergeWorldSettings(ini string, kv map[string]string) string {
	type pair struct{ k, v string }
	var pairs []pair
	seen := map[string]bool{}

	start := strings.Index(ini, "OptionSettings=(")
	end := -1
	if start >= 0 {
		// Find the matching closing ')' — we must skip ')' that appear inside
		// quoted strings (e.g. CrossplayPlatforms="(Steam,Xbox)").
		depth := 0
		inQuote := false
		for i := start + len("OptionSettings=(") - 1; i < len(ini); i++ {
			ch := ini[i]
			if ch == '"' {
				inQuote = !inQuote
			}
			if inQuote {
				continue
			}
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
	}

	if start >= 0 && end > start {
		inner := ini[start+len("OptionSettings=(") : end]
		// Parse key=value pairs respecting quotes (values may contain commas)
		for _, seg := range splitOptionPairs(inner) {
			eq := strings.Index(seg, "=")
			if eq < 0 {
				continue
			}
			k := strings.TrimSpace(seg[:eq])
			v := strings.TrimSpace(seg[eq+1:])
			v = strings.Trim(v, `"`)
			if k != "" {
				pairs = append(pairs, pair{k, v})
				seen[k] = true
			}
		}
	}

	// Override / add new values
	for k, v := range kv {
		if seen[k] {
			for i := range pairs {
				if pairs[i].k == k {
					pairs[i].v = v
					break
				}
			}
		} else {
			pairs = append(pairs, pair{k, v})
			seen[k] = true
		}
	}

	// Rebuild OptionSettings line
	var parts []string
	for _, p := range pairs {
		v := p.v
		if needsQuote(p.k, v) {
			v = `"` + strings.ReplaceAll(v, `"`, `\"`) + `"`
		}
		parts = append(parts, p.k+"="+v)
	}
	newBlock := "OptionSettings=(" + strings.Join(parts, ",") + ")"

	if start >= 0 && end >= 0 {
		return ini[:start] + newBlock + ini[end+1:]
	}
	return strings.TrimRight(ini, "\r\n") + "\n" + newBlock + "\n"
}

func splitOptionPairs(s string) []string {
	var segs []string
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == ',' && !inQuote {
			segs = append(segs, strings.TrimSpace(s[start:i]))
			start = i + 1
		}
	}
	if start < len(s) {
		segs = append(segs, strings.TrimSpace(s[start:]))
	}
	return segs
}

var bareEnums = map[string]bool{
	"none": true, "all": true, "item": true, "itemandequipment": true,
	"text": true, "json": true,
	"casual": true, "normal": true, "hard": true,
	"allowallexceptbanned": true, "allowonlylisted": true,
	"blockimport": true, "clamptomaxvalues": true, "removeFromPal": true,
	"playerdropitem": true,
}

func needsQuote(key, v string) bool {
	bareEmpty := []string{
		"DenyTechnologyList", "CrossplayPlatforms",
	}
	for _, k := range bareEmpty {
		if strings.EqualFold(key, k) {
			return false
		}
	}
	if v == "" {
		return true
	}
	alwaysQuoted := []string{
		"AdminPassword", "ServerPassword", "ServerName", "ServerDescription",
		"PublicIP", "Region", "BanListURL", "RandomizerSeed",
		"AdditionalDropItemWhenPlayerKillingInPvPMode",
	}
	for _, k := range alwaysQuoted {
		if strings.EqualFold(key, k) {
			return true
		}
	}
	lower := strings.ToLower(v)
	if lower == "true" || lower == "false" {
		return false
	}
	if strings.HasPrefix(v, "(") {
		return false
	}
	isNum := true
	for _, ch := range v {
		if (ch < '0' || ch > '9') && ch != '.' && ch != '-' {
			isNum = false
			break
		}
	}
	if isNum {
		return false
	}
	if bareEnums[lower] {
		return false
	}
	return true
}
