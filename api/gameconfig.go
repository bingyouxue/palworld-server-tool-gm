package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
)

// getGameConfig reads a raw config file and returns its content.
// type: "world" -> PalWorldSettings.ini, "engine" -> Engine.ini, "paldefender" -> PalDefender/Config.json
func getGameConfig(c *gin.Context) {
	t := c.Param("type")
	path, err := resolveGameConfigPath(t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"content": "", "path": path, "exists": false})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(data), "path": path, "exists": true})
}

// putGameConfig writes content back to the config file (creates if not exists).
// For world config, also refreshes pending_world_settings_patch so that the
// watch-and-reapply goroutine uses the latest settings if the server is
// (re)started and rewrites the INI with its own defaults.
func putGameConfig(c *gin.Context) {
	t := c.Param("type")
	path, err := resolveGameConfigPath(t)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Keep pending_world_settings_patch in sync so that startServer's
	// watch-and-reapply goroutine always re-applies the most recent world
	// settings rather than an outdated snapshot from the setup wizard.
	if strings.ToLower(t) == "world" {
		kv := parseIniToKV(req.Content)
		if len(kv) > 0 {
			store := config.CurrentStore()
			cfg := store.Config()
			if value := strings.TrimSpace(kv["RCONPort"]); value != "" {
				cfg.Rcon.Address = "127.0.0.1:" + value
			}
			if value := strings.TrimSpace(kv["RESTAPIPort"]); value != "" {
				cfg.Rest.Address = "http://127.0.0.1:" + value
			}
			if password, ok := kv["AdminPassword"]; ok {
				cfg.Rcon.Password = password
				cfg.Rest.Password = password
			}
			if updateErr := store.Update(cfg, ""); updateErr != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "update PST RCON/REST settings: " + updateErr.Error()})
				return
			}
			if patchData, jsonErr := json.Marshal(kv); jsonErr == nil {
				_ = store.SetKV("pending_world_settings_patch", patchData)
				_ = store.SetKV("pending_world_settings_ini", []byte(path))
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "path": path})
}

// parseIniToKV parses the OptionSettings=(…) block of a PalWorldSettings.ini
// string into a flat key→value map. Returns nil if nothing useful is found.
func parseIniToKV(ini string) map[string]string {
	start, end := findOptionSettingsBounds(ini)
	if start < 0 || end < 0 {
		return nil
	}
	inner := ini[start+len("OptionSettings=(") : end]
	result := make(map[string]string)
	for _, seg := range splitOptionPairs(inner) {
		eq := strings.Index(seg, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(seg[:eq])
		v := unquoteWorldSettingValue(strings.TrimSpace(seg[eq+1:]))
		if k != "" {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// resolveGameConfigPath derives the absolute path for the requested config type
// from the configured save path.
// save.path is typically: …/Pal/Saved  or  …/Pal/Saved/SaveGames  or  …/Pal/Binaries/Win64
// We walk up to find the "Pal" folder, then construct the standard sub-path.
func resolveGameConfigPath(t string) (string, error) {
	savePath := config.Current().Save.Path
	if savePath == "" {
		return "", &configPathError{"save.path is not configured"}
	}

	palDir := findPalDir(savePath)
	if palDir == "" {
		// Fall back: assume savePath itself is the Pal root
		palDir = filepath.Clean(savePath)
	}

	switch strings.ToLower(t) {
	case "world":
		if runtime.GOOS == "windows" {
			return filepath.Join(palDir, "Saved", "Config", "WindowsServer", "PalWorldSettings.ini"), nil
		}
		return filepath.Join(palDir, "Saved", "Config", "LinuxServer", "PalWorldSettings.ini"), nil
	case "engine":
		if runtime.GOOS == "windows" {
			return filepath.Join(palDir, "Saved", "Config", "WindowsServer", "Engine.ini"), nil
		}
		return filepath.Join(palDir, "Saved", "Config", "LinuxServer", "Engine.ini"), nil
	case "paldefender":
		// PalDefender's anti-cheat settings live in Config.json.
		if runtime.GOOS == "windows" {
			return filepath.Join(palDir, "Binaries", "Win64", "PalDefender", "Config.json"), nil
		}
		return filepath.Join(palDir, "Binaries", "Linux", "PalDefender", "Config.json"), nil
	case "paldefender-rest":
		// PalDefender v1.8+ split its REST settings into RESTAPI/RESTConfig.json.
		if runtime.GOOS == "windows" {
			return filepath.Join(palDir, "Binaries", "Win64", "PalDefender", "RESTAPI", "RESTConfig.json"), nil
		}
		return filepath.Join(palDir, "Binaries", "Linux", "PalDefender", "RESTAPI", "RESTConfig.json"), nil
	default:
		return "", &configPathError{"unknown config type: " + t}
	}
}

// findPalDir walks up from path to find a directory named "Pal".
func findPalDir(startPath string) string {
	cur := filepath.Clean(startPath)
	for i := 0; i < 10; i++ {
		if strings.EqualFold(filepath.Base(cur), "Pal") {
			return cur
		}
		// Also check if a "Pal" subdirectory exists at this level
		candidate := filepath.Join(cur, "Pal")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return ""
}

type configPathError struct{ msg string }

func (e *configPathError) Error() string { return e.msg }
