package api

import (
	"os"
	"strconv"
	"strings"

	"github.com/zaigie/palworld-server-tool/internal/logger"
)

// serverLaunchArgs builds the command line passed to PalServer on launch.
//
// PalWorldSettings.ini's PublicPort only controls what the server advertises to
// the community server list; the socket it actually binds comes from the
// "-port=" command line argument and falls back to 8211 when omitted. Editing
// PublicPort alone therefore looks like it has no effect, so mirror it onto the
// command line here.
func serverLaunchArgs() []string {
	port := configuredPublicPort()
	if port == 0 {
		return nil
	}
	logger.Infof("[launchServer] applying PublicPort=%d from PalWorldSettings.ini", port)
	return []string{"-port=" + strconv.Itoa(port)}
}

// configuredPublicPort reads PublicPort from PalWorldSettings.ini.
// Returns 0 when the file is missing or the value is absent/invalid, letting
// PalServer apply its own default.
func configuredPublicPort() int {
	path, err := resolveGameConfigPath("world")
	if err != nil {
		logger.Warnf("[launchServer] cannot locate PalWorldSettings.ini: %v", err)
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warnf("[launchServer] read %q: %v", path, err)
		}
		return 0
	}
	kv := parseIniToKV(string(data))
	if len(kv) == 0 {
		return 0
	}
	raw := strings.TrimSpace(kv["PublicPort"])
	if raw == "" {
		return 0
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		logger.Warnf("[launchServer] ignoring invalid PublicPort %q in %s", raw, path)
		return 0
	}
	return port
}
