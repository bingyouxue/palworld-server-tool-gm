package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/database"
	"github.com/zaigie/palworld-server-tool/internal/tool"
	"github.com/zaigie/palworld-server-tool/service"
)

type PlayerOrderBy string

const (
	OrderByLastOnline PlayerOrderBy = "last_online"
	OrderByLevel      PlayerOrderBy = "level"
)

// getPlayerActionUserId 获取用于 kick/ban/unban 操作的 userId
// 优先使用完整的 UserId（支持跨平台），兜底使用 steam_ + SteamId
func getPlayerActionUserId(player database.Player) string {
	if player.UserId != "" {
		return player.UserId
	}
	if player.SteamId != "" {
		return fmt.Sprintf("steam_%s", player.SteamId)
	}
	return ""
}

// listOnlinePlayers godoc
//
//	@Summary		List Online Players
//	@Description	List Online Players
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Success		200	{object}	[]database.OnlinePlayer
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/online_player [get]
func listOnlinePlayers(c *gin.Context) {
	onlinePLayers, err := tool.ShowPlayers()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	service.PutPlayersOnline(database.GetDB(), onlinePLayers)
	// 未登录隐藏敏感字段
	if !c.GetBool("loggedIn") {
		for i := range onlinePLayers {
			onlinePLayers[i].Ip = ""
			if onlinePLayers[i].UserId != "" {
				onlinePLayers[i].UserId = strings.Split(onlinePLayers[i].UserId, "_")[0] + "_"
			}
			onlinePLayers[i].SteamId = ""
		}
	}
	c.JSON(http.StatusOK, onlinePLayers)
}

// putPlayers godoc
//
//	@Summary		Put Players
//	@Description	Put Players Only For SavSync,PlayerSync
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Security		ApiKeyAuth
//
//	@Param			players	body		[]database.Player	true	"Players"
//
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/api/player [put]
func putPlayers(c *gin.Context) {
	var players []database.Player
	if err := c.ShouldBindJSON(&players); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.PutPlayers(database.GetDB(), players); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// listPlayers godoc
//
//	@Summary		List Players
//	@Description	List Players
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Param			order_by	query		PlayerOrderBy	false	"order by field"	enum(last_online,level)
//	@Param			desc		query		bool			false	"order by desc"
//
//	@Success		200			{object}	[]database.TersePlayer
//	@Failure		400			{object}	ErrorResponse
//	@Router			/api/player [get]
func listPlayers(c *gin.Context) {
	orderBy := c.Query("order_by")
	desc := c.Query("desc")
	players, err := service.ListPlayers(database.GetDB())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//未登录隐藏字段
	if !c.GetBool("loggedIn") {
		for i := range players {
			players[i].Ip = ""
			if players[i].UserId != "" {
				players[i].UserId = strings.Split(players[i].UserId, "_")[0] + "_"
			}
			players[i].SteamId = ""
		}
	}
	//排序
	if orderBy == "level" {
		sort.Slice(players, func(i, j int) bool {
			if desc == "true" {
				return players[i].Level > players[j].Level
			}
			return players[i].Level < players[j].Level
		})
	}
	if orderBy == "last_online" {
		sort.Slice(players, func(i, j int) bool {
			if desc == "true" {
				return players[i].LastOnline.Sub(players[j].LastOnline) > 0
			}
			return players[i].LastOnline.Sub(players[j].LastOnline) < 0
		})
	}
	c.JSON(http.StatusOK, players)
}

// getPlayer godoc
//
//	@Summary		Get Player
//	@Description	Get Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	database.Player
//	@Failure		400			{object}	ErrorResponse
//	@Failure		404			{object}	EmptyResponse
//	@Router			/api/player/{player_uid} [get]
func getPlayer(c *gin.Context) {
	player, err := service.GetPlayer(database.GetDB(), c.Param("player_uid"))
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	//未登录隐藏字段
	if !c.GetBool("loggedIn") {
		player.Ip = ""
		if player.UserId != "" {
			player.UserId = strings.Split(player.UserId, "_")[0] + "_"
		}
		player.SteamId = ""
	}
	c.JSON(http.StatusOK, player)
}

// kickPlayer godoc
//
//	@Summary		Kick Player
//	@Description	Kick Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/kick [post]
func kickPlayer(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = tool.KickPlayer(getPlayerActionUserId(player))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// banPlayer godoc
//
//	@Summary		Ban Player
//	@Description	Ban Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/ban [post]
func banPlayer(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = tool.BanPlayer(getPlayerActionUserId(player))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// unbanPlayer godoc
//
//	@Summary		Unban Player
//	@Description	Unban Player
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/unban [post]
func unbanPlayer(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = tool.UnBanPlayer(getPlayerActionUserId(player))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// addWhite godoc
//
//	@Summary		Add White List
//	@Description	Add White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Router			/api/whitelist [post]
func addWhite(c *gin.Context) {
	var player database.PlayerW
	if err := c.ShouldBindJSON(&player); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.AddWhitelist(database.GetDB(), player); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// listWhite godoc
//
//	@Summary		List White List
//	@Description	List White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]database.PlayerW
//	@Failure		400	{object}	ErrorResponse
//	@Router			/api/whitelist [get]
func listWhite(c *gin.Context) {
	players, err := service.ListWhitelist(database.GetDB())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, players)
}

// removeWhite godoc
//
//	@Summary		Remove White List
//	@Description	Remove White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string	true	"Player UID"
//
//	@Success		200			{object}	SuccessResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Router			/api/whitelist [delete]
func removeWhite(c *gin.Context) {
	var player database.PlayerW
	if err := c.ShouldBindJSON(&player); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.RemoveWhitelist(database.GetDB(), player); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// putWhite godoc
//
//	@Summary		Put White List
//	@Description	Put White List
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			players	body		[]database.PlayerW	true	"Players"
//
//	@Success		200		{object}	SuccessResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Router			/api/whitelist [put]
func putWhite(c *gin.Context) {
	var players []database.PlayerW
	if err := c.ShouldBindJSON(&players); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := service.PutWhitelist(database.GetDB(), players); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type GiveItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
	Amount int    `json:"amount" binding:"required,min=1"`
}

// giveItem godoc
//
//	@Summary		Give Item to Player
//	@Description	Give item to player via RCON
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string			true	"Player UID"
//	@Param			body		body		GiveItemRequest	true	"Item info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/give_item [post]
func giveItem(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req GiveItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}
	cmd := fmt.Sprintf("give %s %s %d", userId, req.ItemID, req.Amount)
	response, err := tool.CustomCommand(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

type DeleteItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
	Amount int    `json:"amount" binding:"required,min=1"`
}

// deleteItem godoc
//
//	@Summary		Delete Item from Player
//	@Description	Delete item from player inventory via RCON
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string				true	"Player UID"
//	@Param			body		body		DeleteItemRequest	true	"Item info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/delete_item [post]
func deleteItem(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req DeleteItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}
	cmd := fmt.Sprintf("delitem %s %s %d", userId, req.ItemID, req.Amount)
	response, err := tool.CustomCommand(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

type ReleasePalRequest struct {
	PalType string `json:"pal_type" binding:"required"`
	Level   int    `json:"level"`
	Gender  string `json:"gender"`
	IsLucky bool   `json:"is_lucky"`
}

// releasePal godoc
//
//	@Summary		Release Pal from Player
//	@Description	Release pal from player via RCON deletepals
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string				true	"Player UID"
//	@Param			body		body		ReleasePalRequest	true	"Pal info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/release_pal [post]
func releasePal(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req ReleasePalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}
	filters := []string{fmt.Sprintf("ID %s", req.PalType)}
	if req.Level > 0 {
		filters = append(filters, fmt.Sprintf("Level=%d", req.Level))
	}
	if req.Gender != "" {
		filters = append(filters, fmt.Sprintf("Gender %s", strings.ToLower(req.Gender)))
	}
	if req.IsLucky {
		filters = append(filters, "Lucky true")
	}
	filters = append(filters, "Limit 1")
	cmd := fmt.Sprintf("deletepals %s %s", userId, strings.Join(filters, " "))
	response, err := tool.CustomCommand(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

type GiveExpRequest struct {
	Exp int `json:"exp" binding:"required,min=1"`
}

// giveExp godoc
//
//	@Summary		Give Exp to Player
//	@Description	Give experience points to player via RCON
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string			true	"Player UID"
//	@Param			body		body		GiveExpRequest	true	"Exp info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/give_exp [post]
func giveExp(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req GiveExpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}
	cmd := fmt.Sprintf("give_exp %s %d", userId, req.Exp)
	response, err := tool.CustomCommand(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

type GiveTechPointRequest struct {
	Point int `json:"point" binding:"required,min=1"`
}

// giveTechPoint godoc
//
//	@Summary		Learn Tech for Player
//	@Description	Learn technology for player via RCON /learntech
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string					true	"Player UID"
//	@Param			body		body		GiveTechPointRequest	true	"Tech info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/give_tech_point [post]
func giveTechPoint(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req GiveTechPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}
	cmd := fmt.Sprintf("givetechpoints %s %d", userId, req.Point)
	response, err := tool.CustomCommand(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

// giveAncientTechPoint godoc
//
//	@Summary		Learn All Tech for Player
//	@Description	Learn all technology for player via RCON /learntech all
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string					true	"Player UID"
//	@Param			body		body		GiveTechPointRequest	true	"Tech info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/give_ancient_tech_point [post]
func giveAncientTechPoint(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req GiveTechPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}
	cmd := fmt.Sprintf("givebosstechpoints %s %d", userId, req.Point)
	response, err := tool.CustomCommand(cmd)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}


// ─── Give Custom Pal (givepal_j via PalDefender template) ───────────────────

type GiveCustomPalRequest struct {
	PalID              string            `json:"pal_id" binding:"required"`
	Nickname           string            `json:"nickname"`
	Gender             string            `json:"gender"`
	Level              *int              `json:"level"`
	IsAwakening        bool              `json:"is_awakening"`
	PartnerSkillLevel  *int              `json:"partner_skill_level"`
	Passives           []string          `json:"passives"`
	Skills             []string          `json:"active_skills"`
	Stars              *int              `json:"stars"`
	IVs                struct {
		Health      *int `json:"health"`
		AttackMelee *int `json:"attack_melee"`
		AttackShot  *int `json:"attack_shot"`
		Defense     *int `json:"defense"`
	} `json:"ivs"`
	Souls struct {
		Health     *int `json:"health"`
		Attack     *int `json:"attack"`
		Defense    *int `json:"defense"`
		CraftSpeed *int `json:"craft_speed"`
	} `json:"souls"`
	ExtraWorkSuitabilities map[string]int `json:"extra_work_suitabilities"`
}

// giveCustomPal godoc
//
//	@Summary		Give Custom Pal to Player
//	@Description	Give a customized pal via PalDefender givepal_j RCON command
//	@Tags			Player
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Param			player_uid	path		string				true	"Player UID"
//	@Param			body		body		GiveCustomPalRequest	true	"Pal info"
//	@Success		200			{object}	MessageResponse
//	@Failure		400			{object}	ErrorResponse
//	@Failure		401			{object}	ErrorResponse
//	@Failure		404			{object}	ErrorResponse
//	@Router			/api/player/{player_uid}/give_custom_pal [post]
func giveCustomPal(c *gin.Context) {
	playerUid := c.Param("player_uid")
	player, err := service.GetPlayer(database.GetDB(), playerUid)
	if err != nil {
		if err == service.ErrNoRecord {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req GiveCustomPalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userId := getPlayerActionUserId(player)
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Player has no valid user_id"})
		return
	}

	templatesDir, err := resolvePalDefenderTemplatesDir()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法定位 PalDefender 模板目录: " + err.Error()})
		return
	}

	payload := buildPalTemplatePayload(req)
	templateName := "pst_" + randomHex(6)
	templatePath := filepath.Join(templatesDir, templateName+".json")
	if err := os.WriteFile(templatePath, payload, 0644); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "写入模板文件失败: " + err.Error()})
		return
	}
	cmd := fmt.Sprintf("givepal_j %s %s", userId, templateName)
	response, err := tool.CustomCommand(cmd)
	// PalDefender reads the template asynchronously after RCON returns,
	// so delay deletion to give it time to load the file.
	go func() { time.Sleep(8 * time.Second); os.Remove(templatePath) }()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": response})
}

func resolvePalDefenderTemplatesDir() (string, error) {
	savePath := strings.TrimSpace(config.Current().Save.Path)
	if savePath == "" {
		return "", fmt.Errorf("Save.Path 未配置，请在系统设置中配置存档路径")
	}
	info, err := os.Stat(savePath)
	if err != nil {
		return "", err
	}
	dir := savePath
	if !info.IsDir() {
		dir = filepath.Dir(savePath)
	}
	current := filepath.Clean(dir)
	for i := 0; i < 12; i++ {
		win64 := filepath.Join(current, "Pal", "Binaries", "Win64")
		if fi, e := os.Stat(win64); e == nil && fi.IsDir() {
			if _, e2 := os.Stat(filepath.Join(win64, "PalDefender.dll")); e2 != nil {
				return "", fmt.Errorf("未在 %s 检测到 PalDefender.dll，请先安装 PalDefender", win64)
			}
			templatesDir := filepath.Join(win64, "PalDefender", "Pals", "Templates")
			if err := os.MkdirAll(templatesDir, 0755); err != nil {
				return "", fmt.Errorf("创建模板目录失败: %w", err)
			}
			return templatesDir, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("从 %q 向上未找到服务器根目录（含 Pal/Binaries/Win64）", savePath)
}

func buildPalTemplatePayload(req GiveCustomPalRequest) []byte {
	m := map[string]any{
		"PalID": strings.TrimSpace(req.PalID),
	}
	if v := strings.TrimSpace(req.Nickname); v != "" {
		m["Nickname"] = v
	}
	if v := strings.TrimSpace(req.Gender); v != "" {
		m["Gender"] = v
	}
	if req.Level != nil {
		m["Level"] = *req.Level
	}
	if len(req.Passives) > 0 {
		p := req.Passives
		if len(p) > 8 {
			p = p[:8]
		}
		m["Passives"] = p
	}
	if len(req.Skills) > 0 {
		s := req.Skills
		if len(s) > 3 {
			s = s[:3]
		}
		m["ActiveSkills"] = s
	}
	if req.Stars != nil {
		m["Stars"] = *req.Stars
		m["Star"] = *req.Stars
		m["Rank"] = *req.Stars
		m["CondensedPals"] = *req.Stars
		condense := 0
		switch *req.Stars {
		case 1:
			condense = 4
		case 2:
			condense = 16
		case 3:
			condense = 32
		case 4:
			condense = 64
		}
		m["CondensedCount"] = condense
		m["CondenseCount"] = condense
		if req.PartnerSkillLevel == nil {
			m["PartnerSkillLevel"] = *req.Stars + 1
		}
	}
	ivs := map[string]int{}
	if req.IVs.Health != nil {
		ivs["Health"] = *req.IVs.Health
	}
	if req.IVs.AttackMelee != nil {
		ivs["AttackMelee"] = *req.IVs.AttackMelee
	}
	if req.IVs.AttackShot != nil {
		ivs["AttackShot"] = *req.IVs.AttackShot
	}
	if req.IVs.Defense != nil {
		ivs["Defense"] = *req.IVs.Defense
	}
	if len(ivs) > 0 {
		m["IVs"] = ivs
	}
	souls := map[string]int{}
	if req.Souls.Health != nil {
		souls["Health"] = *req.Souls.Health
	}
	if req.Souls.Attack != nil {
		souls["Attack"] = *req.Souls.Attack
	}
	if req.Souls.Defense != nil {
		souls["Defense"] = *req.Souls.Defense
	}
	if req.Souls.CraftSpeed != nil {
		souls["CraftSpeed"] = *req.Souls.CraftSpeed
	}
	if len(souls) > 0 {
		m["PalSouls"] = souls
	}
	if req.IsAwakening {
		m["IsAwakening"] = true
	}
	if req.PartnerSkillLevel != nil {
		m["PartnerSkillLevel"] = *req.PartnerSkillLevel
	}
	if len(req.ExtraWorkSuitabilities) > 0 {
		m["ExtraWorkSuitabilities"] = req.ExtraWorkSuitabilities
	}
	data, _ := json.MarshalIndent(m, "", "  ")
	return append(data, '\n')
}

func randomHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	var sb strings.Builder
	for _, b := range buf {
		fmt.Fprintf(&sb, "%02x", b)
	}
	return sb.String()
}
