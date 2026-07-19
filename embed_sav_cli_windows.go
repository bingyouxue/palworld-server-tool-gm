//go:build windows

package main

import (
	"crypto/md5"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zaigie/palworld-server-tool/internal/logger"
)

//go:embed sav_cli.exe
var savCliExe embed.FS

func releaseSavCli() {
	data, err := savCliExe.ReadFile("sav_cli.exe")
	if err != nil {
		logger.Errorf("[releaseSavCli] read embedded sav_cli.exe: %v\n", err)
		return
	}
	execPath, err := os.Executable()
	if err != nil {
		logger.Errorf("[releaseSavCli] get executable path: %v\n", err)
		return
	}
	dest := filepath.Join(filepath.Dir(execPath), "sav_cli.exe")
	embeddedMD5 := fmt.Sprintf("%x", md5.Sum(data))
	logger.Infof("[releaseSavCli] embedded sav_cli.exe size=%d md5=%s\n", len(data), embeddedMD5)

	// Compare by MD5 so any content change triggers a re-release.
	if existing, err2 := os.ReadFile(dest); err2 == nil {
		existingMD5 := fmt.Sprintf("%x", md5.Sum(existing))
		logger.Infof("[releaseSavCli] existing sav_cli.exe size=%d md5=%s\n", len(existing), existingMD5)
		if embeddedMD5 == existingMD5 {
			logger.Infof("[releaseSavCli] sav_cli.exe is up to date, skipping release\n")
			return
		}
		logger.Infof("[releaseSavCli] sav_cli.exe changed, replacing...\n")
	}
	if err := os.WriteFile(dest, data, 0755); err != nil {
		logger.Errorf("[releaseSavCli] write sav_cli.exe: %v\n", err)
		return
	}
	logger.Infof("[releaseSavCli] released sav_cli.exe to %s\n", dest)
}
