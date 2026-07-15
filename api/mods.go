package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/mods"
)

var (
	modInstallMu  sync.Mutex
	modInstallChs = map[int]chan string{}
	modInstallIdx int
)

type modInstallRequest struct {
	Component string `json:"component"`
	Channel   string `json:"channel"`
}

func installMod(c *gin.Context) {
	var req modInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comp := mods.Component(req.Component)
	if comp != mods.ComponentPalDefender && comp != mods.ComponentUE4SS {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown component"})
		return
	}
	channel := mods.ChannelStable
	if req.Channel == string(mods.ChannelBeta) {
		channel = mods.ChannelBeta
	}
	serverRoot := serverRootFromSavePath(config.Current().Save.Path)
	if serverRoot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server not found; check save.path"})
		return
	}

	modInstallMu.Lock()
	modInstallIdx++
	id := modInstallIdx
	ch := make(chan string, 128)
	modInstallChs[id] = ch
	modInstallMu.Unlock()

	go func() {
		defer func() {
			modInstallMu.Lock()
			delete(modInstallChs, id)
			modInstallMu.Unlock()
			close(ch)
		}()
		version, err := mods.Install(serverRoot, comp, channel, func(msg string) { ch <- msg })
		if err != nil {
			ch <- "[错误] " + err.Error()
			return
		}
		ch <- fmt.Sprintf("[完成] 已安装 %s %s", string(comp), version)
	}()

	c.JSON(http.StatusOK, gin.H{"install_id": id})
}

func getModInstallProgress(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	if _, err := fmt.Sscan(idStr, &id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	modInstallMu.Lock()
	ch, ok := modInstallChs[id]
	modInstallMu.Unlock()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	if !ok {
		c.SSEvent("error", "job not found or already finished")
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

type modRemoveRequest struct {
	Component string `json:"component"`
}

func removeMod(c *gin.Context) {
	var req modRemoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	comp := mods.Component(req.Component)
	if comp != mods.ComponentPalDefender && comp != mods.ComponentUE4SS {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown component"})
		return
	}
	serverRoot := serverRootFromSavePath(config.Current().Save.Path)
	if serverRoot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "server not found"})
		return
	}
	if err := mods.Remove(serverRoot, comp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
