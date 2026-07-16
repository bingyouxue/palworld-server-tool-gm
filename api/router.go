package api

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/zaigie/palworld-server-tool/internal/auth"
)

type SuccessResponse struct {
	Success bool `json:"success"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type EmptyResponse struct{}

func ignoreLogPrefix(path string) bool {
	prefixes := []string{"/swagger/", "/assets/", "/favicon.ico", "/map"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if !ignoreLogPrefix(param.Path) {
			statusColor := param.StatusCodeColor()
			methodColor := param.MethodColor()
			resetColor := param.ResetColor()
			return fmt.Sprintf("[GIN] %v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				statusColor, param.StatusCode, resetColor,
				param.Latency,
				param.ClientIP,
				methodColor, param.Method, resetColor,
				param.Path,
				param.ErrorMessage,
			)
		}
		return ""
	})
}

func RegisterRouter(r *gin.Engine, onConfigInitialized func()) {
	r.Use(Logger(), gin.Recovery())

	r.POST("/api/login", loginHandler)
	r.GET("/api/config/status", getConfigStatus)
	r.POST("/api/config/initialize", func(c *gin.Context) {
		initializeConfig(c, onConfigInitialized)
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	apiGroup := r.Group("/api")

	anonymousGroup := apiGroup.Group("")
	{
		anonymousGroup.GET("/server", getServer)
		anonymousGroup.GET("/server/tool", getServerTool)
		anonymousGroup.GET("/server/metrics", getServerMetrics)
		anonymousGroup.GET("/server/plugins", getPluginStatus)
		anonymousGroup.GET("/paldefender/version", getPalDefenderVersion)
		anonymousGroup.GET("/guild", listGuilds)
		anonymousGroup.GET("/guild/:admin_player_uid", getGuild)
	}
	// 根据登录状态返回不同结果
	OptionalGroup := apiGroup.Group("")
	OptionalGroup.Use(auth.OptionalJWTMiddleware())
	{
		OptionalGroup.GET("/online_player", listOnlinePlayers)
		OptionalGroup.GET("/player", listPlayers)
		OptionalGroup.GET("/player/:player_uid", getPlayer)
	}

	authGroup := apiGroup.Group("")
	authGroup.Use(auth.JWTAuthMiddleware())
	{
		authGroup.POST("/server/broadcast", publishBroadcast)
		authGroup.POST("/server/shutdown", shutdownServer)
		authGroup.POST("/server/start", startServer)
		authGroup.PUT("/player", putPlayers)
		authGroup.POST("/player/:player_uid/kick", kickPlayer)
		authGroup.POST("/player/:player_uid/ban", banPlayer)
		authGroup.POST("/player/:player_uid/unban", unbanPlayer)
		authGroup.POST("/player/:player_uid/give_item", giveItem)
		authGroup.POST("/player/:player_uid/delete_item", deleteItem)
		authGroup.POST("/player/:player_uid/release_pal", releasePal)
		authGroup.POST("/player/:player_uid/give_exp", giveExp)
		authGroup.POST("/player/:player_uid/give_tech_point", giveTechPoint)
		authGroup.POST("/player/:player_uid/give_ancient_tech_point", giveAncientTechPoint)
		authGroup.POST("/player/:player_uid/learn_tech", learnTech)
		authGroup.POST("/player/:player_uid/give_custom_pal", giveCustomPal)
		authGroup.PUT("/guild", putGuilds)
		authGroup.POST("/sync", syncData)
		authGroup.GET("/whitelist", listWhite)
		authGroup.POST("/whitelist", addWhite)
		authGroup.DELETE("/whitelist", removeWhite)
		authGroup.PUT("/whitelist", putWhite)
		authGroup.GET("/rcon", listRconCommand)
		authGroup.POST("/rcon", addRconCommand)
		authGroup.POST("/rcon/import", importRconCommands)
		authGroup.POST("/rcon/send", sendRconCommand)
		authGroup.POST("/rcon/exec", execRconCommand)
		authGroup.GET("/rcon/tasks", listRconTasks)
		authGroup.POST("/rcon/tasks", addRconTask)
		authGroup.PUT("/rcon/tasks/:uuid", putRconTask)
		authGroup.DELETE("/rcon/tasks/:uuid", removeRconTask)
		authGroup.POST("/rcon/tasks/:uuid/run", runRconTask)
		authGroup.PUT("/rcon/:uuid", putRconCommand)
		authGroup.DELETE("/rcon/:uuid", removeRconCommand)
		authGroup.GET("/backup", listBackups)
		authGroup.GET("/backup/:backup_id", downloadBackup)
		authGroup.DELETE("/backup/:backup_id", deleteBackup)
		authGroup.POST("/backup/:backup_id/restore", restoreBackup)
		authGroup.GET("/config", getConfig)
		authGroup.PUT("/config", putConfig)
		authGroup.GET("/config/directories", listDirectories)
		authGroup.POST("/config/mkdir", createDirectory)
		authGroup.POST("/config/test/save", testSaveConfig)
		authGroup.POST("/config/test/rcon", testRconConfig)
		authGroup.GET("/gameconfig/:type", getGameConfig)
		authGroup.PUT("/gameconfig/:type", putGameConfig)

		// Mods install/remove (PalDefender / UE4SS)
		authGroup.POST("/server/mods/install", installMod)
		authGroup.POST("/server/mods/remove", removeMod)

		// PalDefender config editor
		authGroup.GET("/paldefender/config", getPalDefenderConfig)
		authGroup.PUT("/paldefender/config", putPalDefenderConfig)
		authGroup.GET("/paldefender/import-rules/:name", getPalDefenderImportRule)
		authGroup.PUT("/paldefender/import-rules/:name", putPalDefenderImportRule)
	}

	// SSE progress streams — these use SSEAuthMiddleware which accepts the JWT
	// as either an Authorization header or a ?token= query parameter, because
	// the browser's EventSource API cannot send custom headers.
	sseGroup := apiGroup.Group("")
	sseGroup.Use(auth.SSEAuthMiddleware())
	{
		sseGroup.GET("/server/mods/install/progress/:id", getModInstallProgress)
	}

	// Setup wizard (no auth required — wizard runs before password is set)
	r.GET("/api/setup/status", getSetupStatus)
	r.POST("/api/setup/adopt", postSetupAdopt)
	r.POST("/api/setup/install", postSetupInstall)
	r.GET("/api/setup/install/progress/:id", getSetupInstallProgress)
	r.POST("/api/setup/complete", postSetupComplete)
	r.POST("/api/setup/server-update", postSetupServerUpdate)
	r.PUT("/api/setup/world-settings", putSetupWorldSettings)
}
