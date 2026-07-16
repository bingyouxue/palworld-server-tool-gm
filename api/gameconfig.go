package api

import (
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
	c.JSON(http.StatusOK, gin.H{"success": true, "path": path})
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
		// PalDefender Config.json lives in Binaries/Win64/PalDefender/Config.json
		if runtime.GOOS == "windows" {
			return filepath.Join(palDir, "Binaries", "Win64", "PalDefender", "Config.json"), nil
		}
		return filepath.Join(palDir, "Binaries", "Linux", "PalDefender", "Config.json"), nil
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
