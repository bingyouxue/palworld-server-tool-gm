package api

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/mods"
)

// readPEFileVersion extracts the FileVersion from a Windows PE binary's
// VS_VERSIONINFO resource block.  It walks the raw bytes looking for the
// magic signature 0xFEEF04BD that precedes the VS_FIXEDFILEINFO struct.
// Returns "unknown" on any error or when the signature is not found.
func readPEFileVersion(dllPath string) string {
	f, err := os.Open(dllPath)
	if err != nil {
		return "unknown"
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "unknown"
	}

	// Signature of VS_FIXEDFILEINFO
	sig := []byte{0xBD, 0x04, 0xEF, 0xFE}
	idx := -1
	for i := 0; i < len(data)-len(sig); i++ {
		if data[i] == sig[0] && data[i+1] == sig[1] && data[i+2] == sig[2] && data[i+3] == sig[3] {
			idx = i
			break
		}
	}
	if idx < 0 || idx+52 > len(data) {
		return "unknown"
	}

	// VS_FIXEDFILEINFO layout (little-endian):
	//   offset  0 : DWORD dwSignature
	//   offset  4 : DWORD dwStrucVersion
	//   offset  8 : DWORD dwFileVersionMS
	//   offset 12 : DWORD dwFileVersionLS
	ms := binary.LittleEndian.Uint32(data[idx+8:])
	ls := binary.LittleEndian.Uint32(data[idx+12:])
	return fmt.Sprintf("%d.%d.%d.%d",
		ms>>16, ms&0xFFFF,
		ls>>16, ls&0xFFFF,
	)
}

// getPalDefenderVersion godoc
//
//	@Summary		Get installed PalDefender DLL version + latest GitHub release
//	@Tags			PalDefender
//	@Produce		json
//	@Router			/api/paldefender/version [get]
func getPalDefenderVersion(c *gin.Context) {
	savePath := config.Current().Save.Path
	serverRoot := serverRootFromSavePath(savePath)

	// Installed version: read from DLL PE header first, fall back to marker
	dllVersion := "unknown"
	if serverRoot != "" {
		dllPath := filepath.Join(mods.Win64Dir(serverRoot), "PalDefender.dll")
		if _, err := os.Stat(dllPath); err == nil {
			dllVersion = readPEFileVersion(dllPath)
		}
	}
	markerVersion := ""
	if serverRoot != "" {
		markerVersion = mods.InstalledVersion(serverRoot, mods.ComponentPalDefender)
	}

	// Latest version: proxy GitHub API to avoid CORS from browser
	latestVersion := ""
	latestErr := ""
	type ghRelease struct {
		TagName string `json:"tag_name"`
	}
	req, _ := http.NewRequest("GET",
		"https://api.github.com/repos/Ultimeit/PalDefender/releases/latest", nil)
	req.Header.Set("User-Agent", "pst-gm")
	req.Header.Set("Accept", "application/vnd.github+json")
	if resp, err := http.DefaultClient.Do(req); err == nil {
		defer resp.Body.Close()
		var rel ghRelease
		if body, err := io.ReadAll(resp.Body); err == nil {
			if json.Unmarshal(body, &rel) == nil {
				latestVersion = rel.TagName
			}
		}
	} else {
		latestErr = err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"dll_version":    dllVersion,
		"marker_version": markerVersion,
		"latest_version": latestVersion,
		"latest_error":   latestErr,
		"has_update":     latestVersion != "" && markerVersion != "" && latestVersion != markerVersion,
	})
}

// palDefenderRoot returns the PalDefender directory under Win64.
func palDefenderRoot(savePath string) string {
	dirs := deriveBinaryDirs(savePath)
	for _, d := range dirs {
		candidate := filepath.Join(d, "PalDefender")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	// Return the expected path even if it doesn't exist yet (server not started)
	if len(dirs) > 0 {
		return filepath.Join(dirs[0], "PalDefender")
	}
	return ""
}

// readJSONFile reads and returns raw JSON bytes from a file.
// Returns nil,nil when file does not exist.
func readJSONFile(path string) (json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

// writeJSONFile pretty-prints v to path.
func writeJSONFile(path string, body json.RawMessage) error {
	// validate JSON
	var check interface{}
	if err := json.Unmarshal(body, &check); err != nil {
		return err
	}
	pretty, err := json.MarshalIndent(check, "", "    ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, pretty, 0644)
}

// getPalDefenderConfig godoc
//
//	@Summary		Get PalDefender Config.json
//	@Tags			PalDefender
//	@Produce		json
//	@Router			/api/paldefender/config [get]
func getPalDefenderConfig(c *gin.Context) {
	pdRoot := palDefenderRoot(config.Current().Save.Path)
	if pdRoot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "save.path not configured"})
		return
	}
	path := filepath.Join(pdRoot, "Config.json")
	data, err := readJSONFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if data == nil {
		c.JSON(http.StatusOK, gin.H{"exists": false, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"exists": true, "data": data})
}

// putPalDefenderConfig godoc
//
//	@Summary		Save PalDefender Config.json
//	@Tags			PalDefender
//	@Accept			json
//	@Produce		json
//	@Router			/api/paldefender/config [put]
func putPalDefenderConfig(c *gin.Context) {
	pdRoot := palDefenderRoot(config.Current().Save.Path)
	if pdRoot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "save.path not configured"})
		return
	}
	var body json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := writeJSONFile(filepath.Join(pdRoot, "Config.json"), body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// importRulePath resolves the path for an import rule file.
// name must be one of: default, example
func importRulePath(pdRoot, name string) (string, bool) {
	name = strings.ToLower(name)
	switch name {
	case "default":
		return filepath.Join(pdRoot, "Pals", "ImportRules", "Default.json"), true
	case "example":
		return filepath.Join(pdRoot, "Pals", "ImportRules", "ExampleOverride.json"), true
	}
	return "", false
}

// getPalDefenderImportRule godoc
//
//	@Summary		Get PalDefender import rule (default or example)
//	@Tags			PalDefender
//	@Produce		json
//	@Param			name	path	string	true	"default or example"
//	@Router			/api/paldefender/import-rules/{name} [get]
func getPalDefenderImportRule(c *gin.Context) {
	pdRoot := palDefenderRoot(config.Current().Save.Path)
	if pdRoot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "save.path not configured"})
		return
	}
	path, ok := importRulePath(pdRoot, c.Param("name"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown rule name; use 'default' or 'example'"})
		return
	}
	data, err := readJSONFile(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if data == nil {
		c.JSON(http.StatusOK, gin.H{"exists": false, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"exists": true, "data": data})
}

// putPalDefenderImportRule godoc
//
//	@Summary		Save PalDefender import rule (default or example)
//	@Tags			PalDefender
//	@Accept			json
//	@Produce		json
//	@Param			name	path	string	true	"default or example"
//	@Router			/api/paldefender/import-rules/{name} [put]
func putPalDefenderImportRule(c *gin.Context) {
	pdRoot := palDefenderRoot(config.Current().Save.Path)
	if pdRoot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "save.path not configured"})
		return
	}
	path, ok := importRulePath(pdRoot, c.Param("name"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown rule name; use 'default' or 'example'"})
		return
	}
	var body json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := writeJSONFile(path, body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
