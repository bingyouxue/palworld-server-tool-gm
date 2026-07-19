package tool

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zaigie/palworld-server-tool/internal/auth"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/source"
	"github.com/zaigie/palworld-server-tool/internal/system"
	"github.com/zaigie/palworld-server-tool/service"
	"go.etcd.io/bbolt"
)

type Sturcture struct {
	Players []database.Player `json:"players"`
	Guilds  []database.Guild  `json:"guilds"`
}

func getSavCli() (string, error) {
	savCliPath := config.Current().Save.DecodePath
	if savCliPath == "" || savCliPath == "/path/to/your/sav_cli" {
		ed, err := system.GetExecDir()
		if err != nil {
			logger.Errorf("error getting exec directory: %s", err)
			return "", err
		}
		savCliPath = filepath.Join(ed, "sav_cli")
		if runtime.GOOS == "windows" {
			savCliPath += ".exe"
		}
	}
	if _, err := os.Stat(savCliPath); err != nil {
		return "", err
	}
	return savCliPath, nil
}

func Decode(file string) error {
	savCli, err := getSavCli()
	if err != nil {
		return errors.New("error getting executable path: " + err.Error())
	}

	levelFilePath, err := getFromSource(file, "decode")
	if err != nil {
		return err
	}
	defer os.RemoveAll(filepath.Dir(levelFilePath))

	settings := config.RuntimeWeb()
	baseUrl := fmt.Sprintf("http://127.0.0.1:%d", settings.Port)
	if settings.TLS && settings.PublicURL != "" && !strings.HasSuffix(baseUrl, "/") {
		baseUrl = settings.PublicURL
	}

	requestUrl := fmt.Sprintf("%s/api/", baseUrl)
	tokenString, err := auth.GenerateToken()
	if err != nil {
		return errors.New("error generating token: " + err.Error())
	}
	execArgs := []string{"-f", levelFilePath, "--request", requestUrl, "--token", tokenString}

	logger.Infof("[Decode] sav_cli=%s\n", savCli)
	logger.Infof("[Decode] level_file=%s\n", levelFilePath)
	logger.Infof("[Decode] request_url=%s\n", requestUrl)

	cmd := exec.Command(savCli, execArgs...)

	// Pipe sav_cli stdout and stderr into the PST logger so they appear in the
	// control-panel log file as well as the console.
	cmd.Stdout = &logWriter{prefix: "[sav_cli] "}
	cmd.Stderr = &logWriter{prefix: "[sav_cli|ERR] "}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start sav_cli: %w", err)
	}
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("sav_cli exited with error: %w", err)
	}
	logger.Infof("[Decode] sav_cli completed successfully\n")
	return nil
}

// logWriter adapts a byte-stream (subprocess stdout/stderr) into the PST
// structured logger so sav_cli output is captured in the log file.
type logWriter struct {
	prefix string
	buf    strings.Builder
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		s := w.buf.String()
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			break
		}
		line := s[:idx]
		w.buf.Reset()
		w.buf.WriteString(s[idx+1:])
		if line != "" {
			logger.Infof("%s%s\n", w.prefix, line)
		}
	}
	return len(p), nil
}

func Backup() (string, error) {
	sourcePath := config.Current().Save.Path

	levelFilePath, err := getFromSource(sourcePath, "backup")
	if err != nil {
		return "", err
	}
	savDir := filepath.Dir(levelFilePath)
	defer os.RemoveAll(savDir)

	backupDir, err := GetBackupDir()
	if err != nil {
		return "", fmt.Errorf("failed to get backup directory: %s", err)
	}

	currentTime := time.Now().Format("2006-01-02-15-04-05")
	backupZipFile := filepath.Join(backupDir, fmt.Sprintf("%s.zip", currentTime))
	// Zip the full save directory (Level.sav + Players/*.sav + any sibling files)
	if err = system.ZipDir(savDir, backupZipFile); err != nil {
		return "", fmt.Errorf("failed to create backup zip: %s", err)
	}
	return filepath.Base(backupZipFile), nil
}

func GetBackupDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	backDir := filepath.Join(wd, "backups")
	if err = system.CheckAndCreateDir(backDir); err != nil {
		return "", err
	}
	return backDir, nil
}

func CleanOldBackups(db *bbolt.DB, keepDays int) error {
	backupDir, err := GetBackupDir()
	if err != nil {
		return fmt.Errorf("failed to get backup directory: %s", err)
	}

	deadline := time.Now().AddDate(0, 0, -keepDays)

	backups, err := service.ListBackups(db, time.Time{}, time.Now())
	if err != nil {
		return fmt.Errorf("failed to list backups: %s", err)
	}

	for _, backup := range backups {
		if backup.SaveTime.Before(deadline) {
			err = os.Remove(filepath.Join(backupDir, backup.Path))
			if err != nil {
				if !os.IsNotExist(err) {
					logger.Errorf("failed to delete old backup file %s: %s", backup.Path, err)
				}
			}

			err = service.DeleteBackup(db, backup.BackupId)
			if err != nil {
				logger.Errorf("failed to delete backup record from database: %s", err)
			}
		}
	}

	return nil
}

func getFromSource(file, way string) (string, error) {
	mode := config.Current().Save.SourceMode
	if mode == "agent" {
		if !strings.HasPrefix(file, "http://") && !strings.HasPrefix(file, "https://") {
			return "", errors.New("agent source must be an http:// or https:// pst-agent URL")
		}
		levelFilePath, err := source.DownloadFromHttp(file, way)
		if err != nil {
			return "", errors.New("error downloading file: " + err.Error())
		}
		return levelFilePath, nil
	}
	if mode != "directory" {
		return "", errors.New("unsupported save source mode")
	}
	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		return "", errors.New("directory source must be a local file system path")
	}
	levelFilePath, err := source.CopyFromLocal(file, way)
	if err != nil {
		return "", errors.New("error copying file from directory source: " + err.Error())
	}
	return levelFilePath, nil
}
