import Service from "./service";

class ApiService extends Service {
  async login(param) {
    let data = param;
    return this.fetch(`/api/login`).post(data).json();
  }

  async getConfigStatus() {
    return this.fetch(`/api/config/status`).get().json();
  }

  async initializeConfig(param) {
    return this.fetch(`/api/config/initialize`).post(param).json();
  }

  async getConfig() {
    return this.fetch(`/api/config`).get().json();
  }

  async updateConfig(param) {
    return this.fetch(`/api/config`).put(param).json();
  }

  async listDirectories(path = "") {
    const query = new URLSearchParams({ path }).toString();
    return this.fetch(`/api/config/directories?${query}`).get().json();
  }

  async testSaveConfig(save) {
    return this.fetch(`/api/config/test/save`).post({ save }).json();
  }

  async testRconConfig(rcon) {
    return this.fetch(`/api/config/test/rcon`).post({ rcon }).json();
  }

  async getServerToolInfo() {
    return this.fetch(`/api/server/tool`).get().json();
  }
  async getServerInfo() {
    return this.fetch(`/api/server`).get().json();
  }
  async getServerMetrics() {
    return this.fetch(`/api/server/metrics`).get().json();
  }
  async getServerPlugins() {
    return this.fetch(`/api/server/plugins`).get().json();
  }
  async startServer() {
    return this.fetch(`/api/server/start`).post({}).json();
  }
  async execRconCommand(command) {
    return this.fetch(`/api/rcon/exec`).post({ command }).json();
  }
  async getGameConfig(type) {
    return this.fetch(`/api/gameconfig/${type}`).get().json();
  }
  async putGameConfig(type, content) {
    return this.fetch(`/api/gameconfig/${type}`).put({ content }).json();
  }
  async sendBroadcast(param) {
    let data = param;
    return this.fetch(`/api/server/broadcast`).post(data).json();
  }
  async shutdownServer(param) {
    let data = param;
    return this.fetch(`/api/server/shutdown`).post(data).json();
  }

  async getPlayerList(param) {
    const query = this.generateQuery(param);
    return this.fetch(`/api/player?${query}`).get().json();
  }
  async getOnlinePlayerList() {
    return this.fetch(`/api/online_player`).get().json();
  }
  async getPlayer(param) {
    const { playerUid } = param;
    return this.fetch(`/api/player/${playerUid}`).get().json();
  }
  async kickPlayer(param) {
    const { playerUid } = param;
    return this.fetch(`/api/player/${playerUid}/kick`).post().json();
  }
  async banPlayer(param) {
    const { playerUid } = param;
    return this.fetch(`/api/player/${playerUid}/ban`).post().json();
  }
  async unbanPlayer(param) {
    const { playerUid } = param;
    return this.fetch(`/api/player/${playerUid}/unban`).post().json();
  }
  async giveItem(param) {
    const { playerUid, item_id, amount } = param;
    return this.fetch(`/api/player/${playerUid}/give_item`)
      .post({ item_id, amount })
      .json();
  }
  async deleteItem(param) {
    const { playerUid, item_id, amount } = param;
    return this.fetch(`/api/player/${playerUid}/delete_item`)
      .post({ item_id, amount })
      .json();
  }
  async releasePal(param) {
    const { playerUid, ...body } = param;
    return this.fetch(`/api/player/${playerUid}/release_pal`).post(body).json();
  }
  async giveExp(param) {
    const { playerUid, exp } = param;
    return this.fetch(`/api/player/${playerUid}/give_exp`).post({ exp }).json();
  }
  async giveTechPoint(param) {
    const { playerUid, point } = param;
    return this.fetch(`/api/player/${playerUid}/give_tech_point`).post({ point }).json();
  }
  async giveAncientTechPoint(param) {
    const { playerUid, point } = param;
    return this.fetch(`/api/player/${playerUid}/give_ancient_tech_point`).post({ point }).json();
  }

  async learnTech(param) {
    const { playerUid, tech_id } = param;
    return this.fetch(`/api/player/${playerUid}/learn_tech`).post({ tech_id }).json();
  }

  async giveCustomPal(param) {
    const { playerUid, ...body } = param;
    return this.fetch(`/api/player/${playerUid}/give_custom_pal`).post(body).json();
  }

  async getGuildList() {
    return this.fetch(`/api/guild`).get().json();
  }
  async getGuild(param) {
    const { adminPlayerUid } = param;
    return this.fetch(`/api/guild/${adminPlayerUid}`).get().json();
  }

  async getWhitelist() {
    return this.fetch(`/api/whitelist`).get().json();
  }

  async addWhitelist(param) {
    let data = param;
    return this.fetch(`/api/whitelist`).post(data).json();
  }

  async removeWhitelist(param) {
    let data = param;
    return this.fetch(`/api/whitelist`).delete(data).json();
  }

  async putWhitelist(param) {
    let data = param;
    return this.fetch(`/api/whitelist`).put(data).json();
  }

  async getRconCommands() {
    return this.fetch(`/api/rcon`).get().json();
  }

  async sendRconCommand(param) {
    const { command } = param;
    return this.fetch(`/api/rcon/exec`).post({ command }).json();
  }

  async addRconCommand(param) {
    let data = param;
    return this.fetch(`/api/rcon`).post(data).json();
  }

  async putRconCommand(uuid, param) {
    let data = param;
    return this.fetch(`/api/rcon/${uuid}`).put(data).json();
  }

  async removeRconCommand(uuid) {
    return this.fetch(`/api/rcon/${uuid}`).delete().json();
  }

  async getRconTasks() {
    return this.fetch(`/api/rcon/tasks`).get().json();
  }

  async addRconTask(param) {
    return this.fetch(`/api/rcon/tasks`).post(param).json();
  }

  async putRconTask(uuid, param) {
    return this.fetch(`/api/rcon/tasks/${uuid}`).put(param).json();
  }

  async removeRconTask(uuid) {
    return this.fetch(`/api/rcon/tasks/${uuid}`).delete().json();
  }

  async runRconTask(uuid) {
    return this.fetch(`/api/rcon/tasks/${uuid}/run`).post().json();
  }

  async getBackupList(param) {
    const query = this.generateQuery(param);
    return this.fetch(`/api/backup?${query}`).get().json();
  }

  async removeBackup(uuid) {
    return this.fetch(`/api/backup/${uuid}`).delete().json();
  }

  async downloadBackup(uuid) {
    return this.fetch(`/api/backup/${uuid}`).get().blob();
  }

  async restoreBackup(uuid) {
    return this.fetch(`/api/backup/${uuid}/restore`).post({}).json();
  }

  async getSetupStatus() {
    return this.fetch(`/api/setup/status`).get().json();
  }

  async postSetupAdopt(serverDir) {
    return this.fetch(`/api/setup/adopt`).post({ server_dir: serverDir }).json();
  }

  async postSetupInstall(installDir, steamcmdDir = "") {
    return this.fetch(`/api/setup/install`).post({ install_dir: installDir, steamcmd_dir: steamcmdDir }).json();
  }

  async postSetupComplete() {
    return this.fetch(`/api/setup/complete`).post({}).json();
  }

  async postSetupServerUpdate(serverDir = "") {
    return this.fetch(`/api/setup/server-update`).post({ server_dir: serverDir }).json();
  }

  async getPalDefenderConfig() {
    return this.fetch(`/api/paldefender/config`).get().json();
  }

  async putPalDefenderConfig(data) {
    return this.fetch(`/api/paldefender/config`).put(data).json();
  }

  async getPalDefenderImportRule(name) {
    return this.fetch(`/api/paldefender/import-rules/${name}`).get().json();
  }

  async putPalDefenderImportRule(name, data) {
    return this.fetch(`/api/paldefender/import-rules/${name}`).put(data).json();
  }

  // from: "sav" | "rest"
  async syncData(from) {
    return this.fetch(`/api/sync?from=${from}`).post().json();
  }

}

export default ApiService;
