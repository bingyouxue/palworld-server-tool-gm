package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/logger"
	"github.com/zaigie/palworld-server-tool/internal/system"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

// listBackups godoc
//
//	@Summary		List backups within a specified time range
//	@Description	List all backups or backups within a specific time range.
//	@Tags			backup
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			startTime	query		int	false	"Start time of the backup range in timestamp"
//	@Param			endTime		query		int	false	"End time of the backup range in timestamp"
//	@Success		200			{array}		database.Backup
//	@Failure		400			{object}	ErrorResponse
//	@Router			/api/backup [get]
func listBackups(c *gin.Context) {
	var startTimestamp, endTimestamp int64
	var startTime, endTime time.Time
	var err error

	startTimeStr, endTimeStr := c.Query("startTime"), c.Query("endTime")

	if startTimeStr != "" {
		startTimestamp, err = strconv.ParseInt(startTimeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
			return
		}
		startTime = time.Unix(0, startTimestamp*int64(time.Millisecond))
	}

	if endTimeStr != "" {
		endTimestamp, err = strconv.ParseInt(endTimeStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
			return
		}
		endTime = time.Unix(0, endTimestamp*int64(time.Millisecond))
	}

	backups, err := service.ListBackups(database.GetDB(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backups)
}

// downloadBackup godoc
//
//	@Summary		Download Backup
//	@Description	Download a backup
//	@Tags			backup
//	@Accept			json
//	@Produce		application/octet-stream
//	@Security		ApiKeyAuth
//	@Param			backup_id	path		string	true	"Backup ID"
//	@Success		200			{file}		"Backupfile"
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Failure		500			{object}	ErrorResponse
//	@Router			/api/backup/{backup_id} [get]
func downloadBackup(c *gin.Context) {
	backupId := c.Param("backup_id")
	backup, err := service.GetBackup(database.GetDB(), backupId)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backupDir, err := tool.GetBackupDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", backup.Path))
	c.File(filepath.Join(backupDir, backup.Path))
}

// deleteBackup godoc
//
//	@Summary		Delete Backup
//	@Description	Delete a backup
//	@Tags			backup
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			backup_id	path		string	true	"Backup ID"
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Router			/api/backup/{backup_id} [delete]
func deleteBackup(c *gin.Context) {
	backupId := c.Param("backup_id")
	var backup database.Backup
	backup, err := service.GetBackup(database.GetDB(), backupId)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.DeleteBackup(database.GetDB(), backupId); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	backupDir, err := tool.GetBackupDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Only attempt file removal when the file actually exists; if it has
	// already been deleted manually the record removal above is enough.
	backupFile := filepath.Join(backupDir, backup.Path)
	if rmErr := os.Remove(backupFile); rmErr != nil && !os.IsNotExist(rmErr) {
		c.JSON(http.StatusBadRequest, gin.H{"error": rmErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// restoreBackup godoc
//
//	@Summary		Restore Backup
//	@Description	Stop the game server, restore the backup zip over the live save directory, then restart the server.
//	@Tags			backup
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			backup_id	path		string	true	"Backup ID"
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/backup/{backup_id}/restore [post]
func restoreBackup(c *gin.Context) {
	backupId := c.Param("backup_id")
	backup, err := service.GetBackup(database.GetDB(), backupId)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "备份记录不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	backupDir, err := tool.GetBackupDir()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	backupFile := filepath.Join(backupDir, backup.Path)
	if _, statErr := os.Stat(backupFile); os.IsNotExist(statErr) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件已不存在（" + backup.Path + "），请先重新备份"})
		return
	}

	cfg := config.Current()
	savePath := cfg.Save.Path
	if savePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Save.Path 未配置，无法定位存档目录"})
		return
	}

	// 1. Shut down the game server gracefully (best-effort, ignore errors).
	logger.Info("[restore] sending shutdown to game server")
	_ = tool.Shutdown(0, "服务器正在还原存档，即将重启")

	// Wait for the server to release save files.
	time.Sleep(15 * time.Second)

	// 2. Resolve the target save directory.
	savDir, err := system.GetSavDir(savePath)
	if err != nil {
		// savePath itself may be the save dir if it has no Level.sav yet
		savDir = savePath
	}

	// 3. Extract the backup zip into the save directory (overwrites existing files).
	logger.Infof("[restore] extracting %s -> %s", backupFile, savDir)
	if err := os.MkdirAll(savDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建存档目录失败: " + err.Error()})
		return
	}
	if err := system.UnzipDir(backupFile, savDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解压备份失败: " + err.Error()})
		return
	}

	// 4. Relaunch the server (best-effort).
	exePath := findServerExe(savePath)
	if exePath != "" {
		logger.Infof("[restore] launching %s", exePath)
		if _, err := launchServer(exePath, "silent"); err != nil {
			logger.Errorf("[restore] relaunch failed: %v", err)
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "存档还原成功，但自动重启服务器失败，请手动启动: " + err.Error(),
			})
			return
		}
	} else {
		logger.Warn("[restore] PalServer.exe not found, skipping relaunch")
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "存档还原成功，未找到 PalServer.exe，请手动启动服务器",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "存档还原成功，服务器正在重启"})
}
