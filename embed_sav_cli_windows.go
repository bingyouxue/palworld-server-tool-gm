//go:build windows

package main

import (
	"embed"
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
	if info, err2 := os.Stat(dest); err2 == nil && info.Size() == int64(len(data)) {
		return
	}
	if err := os.WriteFile(dest, data, 0755); err != nil {
		logger.Errorf("[releaseSavCli] write sav_cli.exe: %v\n", err)
		return
	}
	logger.Infof("[releaseSavCli] released sav_cli.exe to %s\n", dest)
}
