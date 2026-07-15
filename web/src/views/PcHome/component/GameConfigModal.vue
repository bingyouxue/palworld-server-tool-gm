<script setup>
import { ref, watch } from "vue";
import { useMessage } from "naive-ui";
import ApiService from "@/service/api";

const props = defineProps({ show: { type: Boolean, default: false } });
const emit = defineEmits(["update:show"]);
const message = useMessage();
const api = new ApiService();

const activeTab = ref("world");
const loading = ref(false);
const saving = ref(false);
const rawContent = ref({ world: "", engine: "", paldefender: "" });
const filePaths  = ref({ world: "", engine: "", paldefender: "" });
const rawMode    = ref({ world: false, engine: false, paldefender: false });

const wf = ref({
  ServerName: "Pal Server", ServerDescription: "", ServerPassword: "",
  AdminPassword: "", ServerPlayerMaxNum: 32, CoopPlayerMaxNum: 4,
  PublicIP: "", PublicPort: 8211,
  bIsMultiplay: false, bShowPlayerList: true, bIsShowJoinLeftMessage: true,
  RESTAPIEnabled: true, RESTAPIPort: 8212, RCONEnabled: true, RCONPort: 25575,
  ChatPostLimitPerMinute: 10, LogFormatType: "Text",
  bIsUseBackupSaveData: true, AutoSaveSpan: 600,
  Region: "", CrossplayPlatforms: "(Steam,Xbox,PS5,Mac)",
  BanListURL: "https://b.palworldgame.com/api/banlist.txt",
  bUseAuth: true, bAllowClientMod: true,
  bEnableVoiceChat: false, VoiceChatMaxVolumeDistance: 3000, VoiceChatZeroVolumeDistance: 15000,
  AutoTransferMasterCheckIntervalSeconds: 3600, AutoTransferMasterThresholdDays: 14,
  ItemContainerForceMarkDirtyInterval: 1, MaxGuildsPerFrame: 10,
  PlayerDataPalStorageUpdateCheckTickInterval: 1,
  DayTimeSpeedRate: 1.0, NightTimeSpeedRate: 1.0, ExpRate: 1.0,
  PlayerDamageRateAttack: 1.0, PlayerDamageRateDefense: 1.0,
  PlayerStomachDecreaceRate: 1.0, PlayerStaminaDecreaceRate: 1.0,
  PlayerAutoHPRegeneRate: 1.0, PlayerAutoHpRegeneRateInSleep: 1.0,
  PalCaptureRate: 1.0, PalAppearRate: 1.0,
  PalDamageRateAttack: 1.0, PalDamageRateDefense: 1.0,
  PalStomachDecreaceRate: 1.0, PalStaminaDecreaceRate: 1.0,
  PalAutoHPRegeneRate: 1.0, PalAutoHpRegeneRateInSleep: 1.0,
  PalEggDefaultHatchingTime: 72.0,
  EnemyDropItemRate: 1.0, WorkSpeedRate: 1.0,
  CollectionDropRate: 1.0, CollectionObjectHpRate: 1.0, CollectionObjectRespawnSpeedRate: 1.0,
  BuildObjectHpRate: 1.0, BuildObjectDamageRate: 1.0, BuildObjectDeteriorationDamageRate: 1.0,
  GuildPlayerMaxNum: 20, BaseCampMaxNum: 128, BaseCampWorkerMaxNum: 15,
  DeathPenalty: "None",
  bEnablePlayerToPlayerDamage: false, bEnableFriendlyFire: false,
  bEnableInvaderEnemy: true, bActiveUNKO: false,
  bEnableAimAssistPad: true, bEnableAimAssistKeyboard: false,
  bHardcore: false, bShowPlayerListOnDeathScreen: true,
  DropItemMaxNum: 3000, DropItemMaxNum_UNKO: 100, DropItemAliveMaxHours: 1.0,
  bAutoResetGuildNoOnlinePlayers: false, AutoResetGuildTimeNoOnlinePlayers: 72.0,
  bIsStartLocationSelectByMap: true, bExistPlayerAfterLogout: false,
  bEnableNonLoginPenalty: true, bEnableFastTravel: true, bIsRandomizerPalAppear: false,
  bEnablePalBox: true, bEnableGlobalPalBox: false,
  AllowGlobalPalboxExport: false, AllowGlobalPalboxImport: false,
});

const ef = ref({
  NetServerMaxTickRate: 120, MaxClientRate: 0, MaxInternetClientRate: 0,
  ConnectionTimeout: 0, InitialConnectTimeout: 0,
  bUseFixedFrameRate: false, FixedFrameRate: 30, bSmoothFrameRate: false,
  gcTimeBetweenPurging: 0,
  useperfthreads: false, NoAsyncLoadingThread: false,
  UseMultithreadForDS: false, NumberOfWorkerThreadsServer: 0,
});

const pd = ref({
  EnableRestAPI: true, RestAPIPort: 17993,
  MOTD: ["Welcome {PlayerName} to {ServerName}!"],
  shouldWarnCheaters: true, shouldWarnCheatersReason: true,
  shouldKickCheaters: false, shouldBanCheaters: false, shouldIPBanCheaters: false,
  steamidProtection: true, blockTowerBossCapture: false,
  disableIllegalItemProtection: false, doActionUponIllegalPalStats: false,
  palStatsMaxRank: -1, pvpMaxToBuildingDamage: 0, pvpMaxToPlayerDamage: 0,
  pvpMaxToPalDamage: 0, pveMaxToPalBanThreshold: 0, treeLimiter: 0,
  useWhitelist: false, useAdminWhitelist: true,
  adminAutoLogin: false, preventAdminPasswordInChat: true,
  allowAdminCheats: true, allowGodmodeOnehit: false,
  chatBypassWait: true, chatMessageMaxLen: 128,
  announceConnections: true, dontAnnounceAdminConnections: true,
  announcePunishments: false, announcePlayerDeaths: false,
  announceOpenOilrigBoxes: false, announceHelicopterKills: false,
  announcePlayerSummons: false, announceAdminSummons: false,
  logChat: true, logRCON: false, logPlayerLogins: true, logPlayerDeaths: true,
  logPlayerBuildings: false, logPlayerSummons: false, logPlayerCaptures: true,
  logCraftings: false, logTechUnlocks: false,
  logPlayerUID: true, logPlayerIP: false, logNetworking: false,
  exitServerOnStartupFailure: true, disableButchering: false,
  disableRenaming: false, disablePalRenaming: false,
  OilrigGoalBoxLocktime: 300, RCONTimeout: 31, RCONUsePacketIdFix: true,
});

// ── INI / JSON helpers ─────────────────────────────────────────────────────
function parsePalWorldSettings(text) {
  const m = text.match(/OptionSettings=\(([^]*)\)/);
  if (!m) return {};
  const inner = m[1];
  const result = {};
  let depth = 0, start = 0;
  for (let i = 0; i <= inner.length; i++) {
    const ch = inner[i];
    if (ch === "(") depth++;
    else if (ch === ")") depth--;
    if ((ch === "," || i === inner.length) && depth === 0) {
      const pair = inner.slice(start, i);
      const eq = pair.indexOf("=");
      if (eq >= 0) result[pair.slice(0, eq).trim()] = pair.slice(eq + 1).trim();
      start = i + 1;
    }
  }
  return result;
}

function applyWorldIni(text) {
  const kv = parsePalWorldSettings(text);
  const f = wf.value;
  Object.keys(f).forEach(k => {
    if (kv[k] === undefined) return;
    const v = kv[k];
    if (typeof f[k] === "boolean") f[k] = v.toLowerCase() === "true";
    else if (typeof f[k] === "number") { const n = parseFloat(v); if (!isNaN(n)) f[k] = n; }
    else f[k] = v.replace(/^"|"$/g, "");
  });
}

function worldToIni() {
  const f = wf.value;
  const pairs = Object.entries(f).map(([k, v]) => {
    if (typeof v === "boolean") return `${k}=${v ? "True" : "False"}`;
    if (typeof v === "number")  return `${k}=${v}`;
    return `${k}="${v}"`;
  });
  return `[/Script/Pal.PalGameWorldSettings]\nOptionSettings=(${pairs.join(",")})\n`;
}

function applyEngineIni(text) {
  const f = ef.value;
  for (const line of text.split(/\r?\n/)) {
    const t = line.trim();
    if (!t || t.startsWith("[") || t.startsWith(";")) continue;
    const eq = t.indexOf("=");
    if (eq < 0) { if (t in f && typeof f[t] === "boolean") f[t] = true; continue; }
    const key = t.slice(0, eq).trim();
    const val = t.slice(eq + 1).trim();
    if (!(key in f)) continue;
    if (typeof f[key] === "boolean") f[key] = val.toLowerCase() === "true";
    else if (typeof f[key] === "number") { const n = parseFloat(val); if (!isNaN(n)) f[key] = n; }
  }
}

function engineToIni() {
  const f = ef.value;
  const lines = [];
  lines.push("[/Script/OnlineSubsystemUtils.IpNetDriver]");
  if (f.NetServerMaxTickRate > 0) lines.push(`NetServerMaxTickRate=${f.NetServerMaxTickRate}`);
  lines.push("", "[/Script/Engine.GameNetworkManager]");
  if (f.MaxClientRate > 0) lines.push(`MaxClientRate=${f.MaxClientRate}`);
  if (f.MaxInternetClientRate > 0) lines.push(`MaxInternetClientRate=${f.MaxInternetClientRate}`);
  if (f.ConnectionTimeout > 0) lines.push(`ConnectionTimeout=${f.ConnectionTimeout}`);
  if (f.InitialConnectTimeout > 0) lines.push(`InitialConnectTimeout=${f.InitialConnectTimeout}`);
  lines.push("", "[/Script/Engine.Engine]");
  if (f.bUseFixedFrameRate) { lines.push("bUseFixedFrameRate=True"); lines.push(`FixedFrameRate=${f.FixedFrameRate}`); }
  if (f.bSmoothFrameRate) lines.push("bSmoothFrameRate=True");
  lines.push("", "[ConsoleVariables]");
  if (f.gcTimeBetweenPurging > 0) lines.push(`gc.TimeBetweenPurgingPendingKillObjects=${f.gcTimeBetweenPurging}`);
  const args = [];
  if (f.useperfthreads) args.push("useperfthreads");
  if (f.NoAsyncLoadingThread) args.push("NoAsyncLoadingThread");
  if (f.UseMultithreadForDS) args.push("UseMultithreadForDS");
  if (f.NumberOfWorkerThreadsServer > 0) args.push(`NumberOfWorkerThreadsServer=${f.NumberOfWorkerThreadsServer}`);
  if (args.length) { lines.push("", "[URL]"); args.forEach(a => lines.push(a)); }
  return lines.join("\n") + "\n";
}

async function loadTab(tab) {
  loading.value = true;
  try {
    const { data } = await api.getGameConfig(tab);
    const d = data.value;
    if (d) {
      rawContent.value[tab] = d.content || "";
      filePaths.value[tab]  = d.path || "";
      if (d.content) {
        if (tab === "world") applyWorldIni(d.content);
        else if (tab === "engine") applyEngineIni(d.content);
        else if (tab === "paldefender") { try { Object.assign(pd.value, JSON.parse(d.content)); } catch {} }
      }
    }
  } catch (e) { console.error("[GameConfig] load:", e); }
  finally { loading.value = false; }
}

async function saveTab(tab) {
  saving.value = true;
  let content = rawMode.value[tab] ? rawContent.value[tab]
    : tab === "world"       ? worldToIni()
    : tab === "engine"      ? engineToIni()
    : JSON.stringify(pd.value, null, 2);
  try {
    const { statusCode, data } = await api.putGameConfig(tab, content);
    if (statusCode.value === 200) {
      message.success("已保存：" + (filePaths.value[tab] || "配置文件"));
      rawContent.value[tab] = content;
    } else { message.error("保存失败：" + (data.value?.error || "未知错误")); }
  } catch (e) { message.error("保存失败：" + e.message); }
  finally { saving.value = false; }
}

watch(() => props.show, v => { if (v) loadTab(activeTab.value); });
watch(activeTab, v => { if (props.show) loadTab(v); });

// ── Field definition arrays ────────────────────────────────────────────────
const W_SERVER_TOGGLES = [
  {k:"bIsMultiplay",l:"开启多人游戏"},{k:"bShowPlayerList",l:"显示玩家列表"},
  {k:"bIsShowJoinLeftMessage",l:"显示加入/离开消息"},{k:"bUseAuth",l:"官方账号验证"},
  {k:"bAllowClientMod",l:"允许客户端模组"},{k:"bIsUseBackupSaveData",l:"启用存档备份"},
  {k:"bEnableVoiceChat",l:"启用语音聊天"},
];
const W_GAMEPLAY_TOGGLES = [
  {k:"bEnablePlayerToPlayerDamage",l:"玩家间伤害 (PvP)"},{k:"bEnableFriendlyFire",l:"友军伤害"},
  {k:"bEnableInvaderEnemy",l:"突袭事件"},{k:"bHardcore",l:"硬核模式"},
  {k:"bActiveUNKO",l:"启用粪便系统"},{k:"bEnableAimAssistPad",l:"手柄辅助瞄准"},
  {k:"bEnableAimAssistKeyboard",l:"键鼠辅助瞄准"},{k:"bIsStartLocationSelectByMap",l:"出生点地图选择"},
  {k:"bExistPlayerAfterLogout",l:"下线后角色留存"},{k:"bEnableNonLoginPenalty",l:"离线惩罚"},
  {k:"bEnableFastTravel",l:"快速传送"},{k:"bAutoResetGuildNoOnlinePlayers",l:"无人时自动解散公会"},
  {k:"bEnablePalBox",l:"启用帕鲁箱"},{k:"bEnableGlobalPalBox",l:"启用全球帕鲁箱"},
  {k:"AllowGlobalPalboxExport",l:"允许导出全球帕鲁箱"},{k:"AllowGlobalPalboxImport",l:"允许导入全球帕鲁箱"},
  {k:"bShowPlayerListOnDeathScreen",l:"死亡界面显示玩家列表"},{k:"bIsRandomizerPalAppear",l:"帕鲁随机刷新"},
];
const W_RATE_FIELDS = [
  {k:"DayTimeSpeedRate",l:"白天速度",mn:0.1,mx:5,st:0.1},
  {k:"NightTimeSpeedRate",l:"夜晚速度",mn:0.1,mx:5,st:0.1},
  {k:"ExpRate",l:"经验倍率",mn:0.1,mx:20,st:0.1},
  {k:"PalCaptureRate",l:"捕获率倍率",mn:0.1,mx:5,st:0.1},
  {k:"PalAppearRate",l:"帕鲁出现率",mn:0.1,mx:5,st:0.1},
  {k:"PalDamageRateAttack",l:"帕鲁攻击倍率",mn:0.1,mx:20,st:0.1},
  {k:"PalDamageRateDefense",l:"帕鲁防御倍率",mn:0.1,mx:20,st:0.1},
  {k:"PalStomachDecreaceRate",l:"帕鲁饱食度消耗",mn:0.1,mx:5,st:0.1},
  {k:"PalStaminaDecreaceRate",l:"帕鲁体力消耗",mn:0.1,mx:5,st:0.1},
  {k:"PalAutoHPRegeneRate",l:"帕鲁自动恢复HP",mn:0.1,mx:5,st:0.1},
  {k:"PalAutoHpRegeneRateInSleep",l:"帕鲁睡眠恢复HP",mn:0.1,mx:5,st:0.1},
  {k:"EnemyDropItemRate",l:"掉落率倍率",mn:0.1,mx:20,st:0.1},
  {k:"WorkSpeedRate",l:"工作速度倍率",mn:0.1,mx:20,st:0.1},
  {k:"CollectionDropRate",l:"采集掉落倍率",mn:0.1,mx:20,st:0.1},
  {k:"CollectionObjectHpRate",l:"采集对象耐久",mn:0.1,mx:20,st:0.1},
  {k:"CollectionObjectRespawnSpeedRate",l:"采集复生速度",mn:0.1,mx:20,st:0.1},
  {k:"PlayerDamageRateAttack",l:"玩家攻击倍率",mn:0.1,mx:20,st:0.1},
  {k:"PlayerDamageRateDefense",l:"玩家防御倍率",mn:0.1,mx:20,st:0.1},
  {k:"PlayerStomachDecreaceRate",l:"玩家饱食度消耗",mn:0.1,mx:5,st:0.1},
  {k:"PlayerStaminaDecreaceRate",l:"玩家体力消耗",mn:0.1,mx:5,st:0.1},
  {k:"PlayerAutoHPRegeneRate",l:"玩家自动恢复HP",mn:0.1,mx:5,st:0.1},
  {k:"PlayerAutoHpRegeneRateInSleep",l:"玩家睡眠恢复HP",mn:0.1,mx:5,st:0.1},
  {k:"BuildObjectHpRate",l:"建筑血量倍率",mn:0.1,mx:20,st:0.1},
  {k:"BuildObjectDamageRate",l:"建筑受损倍率",mn:0.1,mx:20,st:0.1},
  {k:"BuildObjectDeteriorationDamageRate",l:"建筑劣化速度",mn:0.1,mx:20,st:0.1},
  {k:"PalEggDefaultHatchingTime",l:"帕鲁蛋孵化(h)",mn:0,mx:720,st:1},
];
</script>

<template>
  <n-modal :show="show" preset="card" title="配置设定"
    style="width:96%;max-width:860px;max-height:92vh"
    content-style="padding:0;overflow:hidden"
    :bordered="false"
    @update:show="emit('update:show',$event)">

    <n-tabs v-model:value="activeTab" type="line" animated justify-content="space-evenly">

      <!-- ════════════════════════════ 世界设定 ════════════════════════════ -->
      <n-tab-pane name="world" tab="世界设定">
        <n-scrollbar style="max-height:calc(92vh - 155px)">
          <div class="gcfg-pane" v-if="!rawMode.world">
            <n-spin :show="loading">

              <!-- 服务器基础 -->
              <n-card size="small" class="mb-3" title="服务器基础">
                <n-grid :cols="2" :x-gap="12" :y-gap="8">
                  <n-gi><n-form-item label="服务器名称" label-placement="top" size="small">
                    <n-input v-model:value="wf.ServerName" size="small"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="服务器描述" label-placement="top" size="small">
                    <n-input v-model:value="wf.ServerDescription" size="small"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="服务器密码" label-placement="top" size="small">
                    <n-input v-model:value="wf.ServerPassword" type="password" show-password-on="click" size="small"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="管理员密码" label-placement="top" size="small">
                    <n-input v-model:value="wf.AdminPassword" type="password" show-password-on="click" size="small"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="最大玩家数 (ServerPlayerMaxNum)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.ServerPlayerMaxNum" :min="1" :max="32" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="合作玩家数 (CoopPlayerMaxNum)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.CoopPlayerMaxNum" :min="1" :max="8" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="服务器端口 (PublicPort)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.PublicPort" :min="1024" :max="65535" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="公开 IP (PublicIP)" label-placement="top" size="small">
                    <n-input v-model:value="wf.PublicIP" size="small" placeholder="留空自动检测"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="服务器地区 (Region)" label-placement="top" size="small">
                    <n-input v-model:value="wf.Region" size="small" placeholder="留空"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="跨平台连线 (CrossplayPlatforms)" label-placement="top" size="small">
                    <n-input v-model:value="wf.CrossplayPlatforms" size="small"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="封禁名单 URL (BanListURL)" label-placement="top" size="small">
                    <n-input v-model:value="wf.BanListURL" size="small"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="每分钟聊天上限 (ChatPostLimitPerMinute)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.ChatPostLimitPerMinute" :min="1" :max="120" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="日志格式 (LogFormatType)" label-placement="top" size="small">
                    <n-select v-model:value="wf.LogFormatType" size="small" :options="[{label:'文字',value:'Text'},{label:'JSON',value:'Json'}]"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="自动存档间隔秒 (AutoSaveSpan)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.AutoSaveSpan" :min="10" :max="600" :step="5" size="small" style="width:100%"/></n-form-item></n-gi>
                </n-grid>
                <n-grid :cols="2" :x-gap="12" :y-gap="2" class="mt-2">
                  <n-gi v-for="sw in W_SERVER_TOGGLES" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                      <span class="gcfg-label">{{ sw.l }}</span>
                      <n-switch v-model:value="wf[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- REST API / RCON -->
              <n-card size="small" class="mb-3" title="REST API / RCON">
                <n-grid :cols="2" :x-gap="12" :y-gap="4">
                  <n-gi><n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                    <span class="gcfg-label">启用 REST API</span>
                    <n-switch v-model:value="wf.RESTAPIEnabled" size="small"/></n-flex></n-gi>
                  <n-gi><n-form-item label="REST API 端口" label-placement="left" label-width="110" size="small">
                    <n-input-number v-model:value="wf.RESTAPIPort" :min="1024" :max="65535" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                    <span class="gcfg-label">启用 RCON</span>
                    <n-switch v-model:value="wf.RCONEnabled" size="small"/></n-flex></n-gi>
                  <n-gi><n-form-item label="RCON 端口" label-placement="left" label-width="110" size="small">
                    <n-input-number v-model:value="wf.RCONPort" :min="1024" :max="65535" size="small" style="width:100%"/></n-form-item></n-gi>
                </n-grid>
              </n-card>

              <!-- 语音聊天 -->
              <n-card size="small" class="mb-3" title="语音聊天">
                <n-grid :cols="2" :x-gap="12" :y-gap="6">
                  <n-gi><n-form-item label="最大可听距离 (VoiceChatMaxVolumeDistance)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.VoiceChatMaxVolumeDistance" :min="0" :max="50000" :step="100" size="small" style="width:100%" :disabled="!wf.bEnableVoiceChat"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="零音量距离 (VoiceChatZeroVolumeDistance)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.VoiceChatZeroVolumeDistance" :min="0" :max="50000" :step="100" size="small" style="width:100%" :disabled="!wf.bEnableVoiceChat"/></n-form-item></n-gi>
                </n-grid>
              </n-card>

              <!-- 游戏规则 -->
              <n-card size="small" class="mb-3" title="游戏规则">
                <n-form-item label="死亡惩罚 (DeathPenalty)" label-placement="top" size="small" class="mb-2">
                  <n-select v-model:value="wf.DeathPenalty" size="small" style="width:220px"
                    :options="[{label:'无',value:'None'},{label:'仅道具',value:'Item'},{label:'道具+装备',value:'ItemAndEquipment'},{label:'全部',value:'All'}]"/>
                </n-form-item>
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in W_GAMEPLAY_TOGGLES" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                      <span class="gcfg-label">{{ sw.l }}</span>
                      <n-switch v-model:value="wf[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
                <n-grid :cols="3" :x-gap="12" :y-gap="6" class="mt-3">
                  <n-gi><n-form-item label="最大掉落物数" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.DropItemMaxNum" :min="0" :max="10000" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="掉落物存活时间(h)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.DropItemAliveMaxHours" :min="0" :max="24" :step="0.5" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="公会无人解散时间(h)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.AutoResetGuildTimeNoOnlinePlayers" :min="0" :max="720" size="small" style="width:100%"/></n-form-item></n-gi>
                </n-grid>
              </n-card>

              <!-- 倍率 -->
              <n-card size="small" class="mb-3" title="倍率设定">
                <n-grid :cols="2" :x-gap="16" :y-gap="4">
                  <n-gi v-for="field in W_RATE_FIELDS" :key="field.k">
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label">
                        <span>{{ field.l }}</span>
                        <span class="gcfg-rate-key">{{ field.k }}</span>
                      </div>
                      <n-input-number v-model:value="wf[field.k]" :min="field.mn" :max="field.mx" :step="field.st" size="small" style="width:100%"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 公会/基地 -->
              <n-card size="small" class="mb-3" title="公会 / 基地">
                <n-grid :cols="3" :x-gap="12" :y-gap="6">
                  <n-gi><n-form-item label="公会最大玩家" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.GuildPlayerMaxNum" :min="1" :max="100" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="基地数量上限" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.BaseCampMaxNum" :min="1" :max="512" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="基地工人上限" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.BaseCampWorkerMaxNum" :min="1" :max="50" size="small" style="width:100%"/></n-form-item></n-gi>
                </n-grid>
              </n-card>

              <!-- 服务器性能微调 -->
              <n-card size="small" class="mb-3" title="服务器性能微调">
                <n-grid :cols="2" :x-gap="12" :y-gap="6">
                  <n-gi><n-form-item label="物品自动转移检查间隔(s)" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.AutoTransferMasterCheckIntervalSeconds" :min="60" :max="86400" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="物品自动转移天数门槛" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.AutoTransferMasterThresholdDays" :min="0" :max="365" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="物品容器强制同步间隔" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.ItemContainerForceMarkDirtyInterval" :min="0" :max="60" :step="0.1" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="每帧处理公会数量上限" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.MaxGuildsPerFrame" :min="1" :max="100" size="small" style="width:100%"/></n-form-item></n-gi>
                  <n-gi><n-form-item label="帕鲁箱更新检查间隔" label-placement="top" size="small">
                    <n-input-number v-model:value="wf.PlayerDataPalStorageUpdateCheckTickInterval" :min="0" :max="60" :step="0.1" size="small" style="width:100%"/></n-form-item></n-gi>
                </n-grid>
              </n-card>

            </n-spin>
          </div>
          <div v-else class="gcfg-pane">
            <n-input v-model:value="rawContent.world" type="textarea" :rows="30" style="font-family:monospace;font-size:12px"/>
          </div>
        </n-scrollbar>
      </n-tab-pane>

      <!-- ════════════════════════════ 引擎调优 ════════════════════════════ -->
      <n-tab-pane name="engine" tab="引擎调优">
        <n-scrollbar style="max-height:calc(92vh - 155px)">
          <div class="gcfg-pane" v-if="!rawMode.engine">
            <n-spin :show="loading">
              <n-alert type="warning" size="small" :show-icon="true" class="mb-3">
                Engine.ini 修改直接影响服务器性能，改完请观察服务器 FPS。不确定时使用推荐预设。
              </n-alert>
              <n-card size="small" class="mb-3" title="性能预设">
                <n-flex gap="8" wrap>
                  <n-button size="small" @click="Object.assign(ef,{NetServerMaxTickRate:0,MaxClientRate:0,MaxInternetClientRate:0,ConnectionTimeout:0,InitialConnectTimeout:0,bUseFixedFrameRate:false,bSmoothFrameRate:false,gcTimeBetweenPurging:0,useperfthreads:false,NoAsyncLoadingThread:false,UseMultithreadForDS:false,NumberOfWorkerThreadsServer:0})">游戏默认</n-button>
                  <n-button size="small" type="primary" @click="Object.assign(ef,{NetServerMaxTickRate:60,MaxClientRate:104857600,MaxInternetClientRate:104857600,ConnectionTimeout:120,InitialConnectTimeout:120})">平衡(推荐)</n-button>
                  <n-button size="small" type="warning" @click="Object.assign(ef,{NetServerMaxTickRate:90,MaxClientRate:104857600,MaxInternetClientRate:104857600,ConnectionTimeout:120,InitialConnectTimeout:120})">高性能</n-button>
                </n-flex>
              </n-card>
              <n-card size="small" class="mb-3" title="网络">
                <n-grid :cols="1" :y-gap="4">
                  <n-gi v-for="nf in [
                    {k:'NetServerMaxTickRate',l:'服务器 Tick 率 (NetServerMaxTickRate)',note:'每秒更新次数，提高更流畅但耗 CPU；0 = 不设置',min:0,max:120},
                    {k:'MaxClientRate',l:'每玩家带宽上限 bytes/s (MaxClientRate)',note:'提高可减少高 Tick 率下的封包裁切；0 = 不设置',min:0,max:209715200},
                    {k:'MaxInternetClientRate',l:'互联网玩家带宽上限 (MaxInternetClientRate)',note:'同上，用于非局域网玩家；0 = 不设置',min:0,max:209715200},
                    {k:'ConnectionTimeout',l:'连接超时(秒) (ConnectionTimeout)',note:'服务器卡顿时调高可避免误踢玩家；0 = 不设置',min:0,max:600},
                    {k:'InitialConnectTimeout',l:'初次连接超时(秒) (InitialConnectTimeout)',note:'地图大、载入慢时调高；0 = 不设置',min:0,max:600},
                  ]" :key="nf.k">
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label">
                        <span>{{ nf.l }}</span>
                        <span v-if="nf.note" class="gcfg-rate-key">{{ nf.note }}</span>
                      </div>
                      <n-input-number v-model:value="ef[nf.k]" :min="nf.min" :max="nf.max" size="small" style="width:200px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>
              <n-card size="small" class="mb-3" title="帧率">
                <n-grid :cols="2" :x-gap="12" :y-gap="4">
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>固定帧率 (bUseFixedFrameRate)</span><span class="gcfg-rate-key">让服务器以固定步进运行，模拟较稳定</span></div>
                      <n-switch v-model:value="ef.bUseFixedFrameRate" size="small"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>固定帧率值 (FixedFrameRate)</span><span class="gcfg-rate-key">仅在固定帧率开启时生效，通常设为与 Tick 率相同</span></div>
                      <n-input-number v-model:value="ef.FixedFrameRate" :min="20" :max="120" size="small" style="width:140px" :disabled="!ef.bUseFixedFrameRate"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>平滑帧率 (bSmoothFrameRate)</span><span class="gcfg-rate-key">让帧时间变化较平缓，减少突发顿挫</span></div>
                      <n-switch v-model:value="ef.bSmoothFrameRate" size="small"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>
              <n-card size="small" class="mb-3" title="内存">
                <div class="gcfg-rate-item">
                  <div class="gcfg-rate-label">
                    <span>垃圾回收间隔(秒) (gc.TimeBetweenPurgingPendingKillObjects)</span>
                    <span class="gcfg-rate-key">调短压低内存占用但每次回收有短暂顿挫；调长则相反；0 = 不设置</span>
                  </div>
                  <n-input-number v-model:value="ef.gcTimeBetweenPurging" :min="0" :max="600" size="small" style="width:160px"/>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="启动参数（多线程）">
                <n-alert type="warning" size="small" :show-icon="true" class="mb-3">
                  官方注记：v1.0 之后不设置这三个多线程旗标反而可能效能更好——非必开，建议实测比较。
                </n-alert>
                <n-grid :cols="2" :x-gap="12" :y-gap="4">
                  <n-gi v-for="sw in [
                    {k:'useperfthreads',l:'useperfthreads（性能线程）'},
                    {k:'NoAsyncLoadingThread',l:'NoAsyncLoadingThread（停用异步载入线程）'},
                    {k:'UseMultithreadForDS',l:'UseMultithreadForDS（服务器多线程）'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                      <span class="gcfg-label">{{ sw.l }}</span>
                      <n-switch v-model:value="ef[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>工作线程数 (NumberOfWorkerThreadsServer)</span><span class="gcfg-rate-key">0 = 不指定（交给系统）；需搭配上面的多线程旗标</span></div>
                      <n-input-number v-model:value="ef.NumberOfWorkerThreadsServer" :min="0" :max="128" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>
            </n-spin>
          </div>
          <div v-else class="gcfg-pane">
            <n-input v-model:value="rawContent.engine" type="textarea" :rows="30" style="font-family:monospace;font-size:12px"/>
          </div>
        </n-scrollbar>
      </n-tab-pane>

      <!-- ══════════════════════════ PalDefender ══════════════════════════ -->
      <n-tab-pane name="paldefender" tab="PalDefender">
        <n-scrollbar style="max-height:calc(92vh - 155px)">
          <div class="gcfg-pane" v-if="!rawMode.paldefender">
            <n-spin :show="loading">

              <!-- REST API -->
              <n-card size="small" class="mb-3" title="REST API">
                <n-grid :cols="2" :x-gap="12" :y-gap="4">
                  <n-gi><n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                    <span class="gcfg-label">启用 REST API (EnableRestAPI)</span>
                    <n-switch v-model:value="pd.EnableRestAPI" size="small"/>
                  </n-flex></n-gi>
                  <n-gi><n-form-item label="REST API 端口 (RestAPIPort)" label-placement="left" label-width="190" size="small">
                    <n-input-number v-model:value="pd.RestAPIPort" :min="1024" :max="65535" size="small" style="width:100%"/>
                  </n-form-item></n-gi>
                </n-grid>
                <n-text depth="3" style="font-size:11px">启用后可在玩家详情页查看帕鲁与背包。变更需重启服务器生效。</n-text>
              </n-card>

              <!-- MOTD -->
              <n-card size="small" class="mb-3" title="登入公告 (MOTD)">
                <n-text depth="3" style="font-size:11px" class="mb-2 block">玩家加入时显示，每行一条。变更需重启或 reloadcfg 生效。</n-text>
                <n-input
                  :value="Array.isArray(pd.MOTD)?pd.MOTD.join('\n'):(pd.MOTD||'')"
                  @update:value="v => pd.MOTD = v.split('\n')"
                  type="textarea" :rows="4" size="small"
                  placeholder="Welcome {PlayerName} to {ServerName}!"/>
              </n-card>

              <!-- 反外挂处置 -->
              <n-card size="small" class="mb-3" title="反外挂处置">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'shouldWarnCheaters',l:'警告作弊者',d:'检测到作弊时发送警告消息给该玩家'},
                    {k:'shouldWarnCheatersReason',l:'警告时附上原因'},
                    {k:'shouldKickCheaters',l:'自动踢出作弊者'},
                    {k:'shouldBanCheaters',l:'自动封禁作弊者'},
                    {k:'shouldIPBanCheaters',l:'自动 IP 封禁作弊者',d:'IP 封禁可能误伤同一 IP 下的其他玩家'},
                  ]" :key="sw.k">
                    <div class="gcfg-rate-item" style="padding:4px 0">
                      <div class="gcfg-rate-label"><span>{{ sw.l }}</span><span v-if="sw.d" class="gcfg-rate-key">{{ sw.d }}</span></div>
                      <n-switch v-model:value="pd[sw.k]" size="small"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 漏洞防护 -->
              <n-card size="small" class="mb-3" title="漏洞防护">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'steamidProtection',l:'防止重复 UserId 登入'},
                    {k:'blockTowerBossCapture',l:'禁止捕捉塔主'},
                    {k:'disableIllegalItemProtection',l:'停用非法道具防护',d:'关闭后模组/调试道具将不再被拦截，一般不建议停用'},
                    {k:'doActionUponIllegalPalStats',l:'自动处理异常帕鲁数值'},
                  ]" :key="sw.k">
                    <div class="gcfg-rate-item" style="padding:4px 0">
                      <div class="gcfg-rate-label"><span>{{ sw.l }}</span><span v-if="sw.d" class="gcfg-rate-key">{{ sw.d }}</span></div>
                      <n-switch v-model:value="pd[sw.k]" size="small"/>
                    </div>
                  </n-gi>
                </n-grid>
                <n-grid :cols="2" :x-gap="12" :y-gap="6" class="mt-3">
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>帕鲁强化上限 (palStatsMaxRank)</span><span class="gcfg-rate-key">-1 = 自动检测</span></div>
                      <n-input-number v-model:value="pd.palStatsMaxRank" :min="-1" :max="20" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>砍树速率限制 (treeLimiter) 秒/棵</span><span class="gcfg-rate-key">限制每棵树最短破坏时间，避免火箭砍树卡顿；0 = 关闭</span></div>
                      <n-input-number v-model:value="pd.treeLimiter" :min="0" :max="5" :step="0.05" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>PvP 对建筑最大伤害 (pvpMaxToBuildingDamage)</span><span class="gcfg-rate-key">0 = 不限</span></div>
                      <n-input-number v-model:value="pd.pvpMaxToBuildingDamage" :min="0" :max="100000" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>PvP 对玩家最大伤害 (pvpMaxToPlayerDamage)</span><span class="gcfg-rate-key">0 = 不限；Beta 功能</span></div>
                      <n-input-number v-model:value="pd.pvpMaxToPlayerDamage" :min="0" :max="100000" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>PvP 对帕鲁最大伤害 (pvpMaxToPalDamage)</span><span class="gcfg-rate-key">0 = 不限</span></div>
                      <n-input-number v-model:value="pd.pvpMaxToPalDamage" :min="0" :max="100000" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>PvE 帕鲁伤害封禁阈值 (pveMaxToPalBanThreshold)</span><span class="gcfg-rate-key">0 = 关闭</span></div>
                      <n-input-number v-model:value="pd.pveMaxToPalBanThreshold" :min="0" :max="1000000" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 白名单与管理 -->
              <n-card size="small" class="mb-3" title="白名单与管理员">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'useWhitelist',l:'启用白名单 (WhiteList.json)'},
                    {k:'useAdminWhitelist',l:'启用管理员 IP 白名单',d:'需在 adminIPs 中设置 IP（用原始文件编辑）'},
                    {k:'adminAutoLogin',l:'白名单管理员加入时自动登入管理模式'},
                    {k:'preventAdminPasswordInChat',l:'防止管理员密码在聊天中外泄'},
                    {k:'allowAdminCheats',l:'允许管理员使用作弊指令 (godmode 等)'},
                    {k:'allowGodmodeOnehit',l:'godmode 可一击击杀'},
                  ]" :key="sw.k">
                    <div class="gcfg-rate-item" style="padding:4px 0">
                      <div class="gcfg-rate-label"><span>{{ sw.l }}</span><span v-if="sw.d" class="gcfg-rate-key">{{ sw.d }}</span></div>
                      <n-switch v-model:value="pd[sw.k]" size="small"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 聊天 -->
              <n-card size="small" class="mb-3" title="聊天">
                <n-grid :cols="2" :x-gap="12" :y-gap="4">
                  <n-gi><n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                    <span class="gcfg-label">移除聊天冷却时间 (chatBypassWait)</span>
                    <n-switch v-model:value="pd.chatBypassWait" size="small"/>
                  </n-flex></n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>聊天消息长度上限 (chatMessageMaxLen)</span></div>
                      <n-input-number v-model:value="pd.chatMessageMaxLen" :min="1" :max="1000" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 公告 -->
              <n-card size="small" class="mb-3" title="公告">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'announceConnections',l:'公告玩家上下线'},
                    {k:'dontAnnounceAdminConnections',l:'不公告管理员上下线'},
                    {k:'announcePunishments',l:'公告作弊处罚(踢出/封禁)'},
                    {k:'announcePlayerDeaths',l:'公告玩家死亡'},
                    {k:'announceOpenOilrigBoxes',l:'公告钻油平台宝箱开启'},
                    {k:'announceHelicopterKills',l:'公告直升机击杀'},
                    {k:'announcePlayerSummons',l:'公告玩家召唤帕鲁'},
                    {k:'announceAdminSummons',l:'公告管理员召唤帕鲁'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                      <span class="gcfg-label">{{ sw.l }}</span>
                      <n-switch v-model:value="pd[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 日志 -->
              <n-card size="small" class="mb-3" title="日志">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'logChat',l:'记录聊天消息'},
                    {k:'logRCON',l:'记录 RCON 指令使用'},
                    {k:'logPlayerLogins',l:'记录玩家上下线'},
                    {k:'logPlayerDeaths',l:'记录玩家死亡'},
                    {k:'logPlayerBuildings',l:'记录玩家建筑(建造/取消/拆除)'},
                    {k:'logPlayerSummons',l:'记录玩家召唤帕鲁'},
                    {k:'logPlayerCaptures',l:'记录玩家捕捉帕鲁'},
                    {k:'logCraftings',l:'记录玩家制作'},
                    {k:'logTechUnlocks',l:'记录科技解锁'},
                    {k:'logPlayerUID',l:'日志中记录玩家 UserId'},
                    {k:'logPlayerIP',l:'日志中记录玩家 IP'},
                    {k:'logNetworking',l:'记录客户端网络数据'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="gcfg-toggle-row">
                      <span class="gcfg-label">{{ sw.l }}</span>
                      <n-switch v-model:value="pd[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <!-- 其他 -->
              <n-card size="small" class="mb-3" title="其他">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'exitServerOnStartupFailure',l:'PalDefender 启动失败时关闭服务器',d:'保护存档不在无反外挂的情况下运行'},
                    {k:'disableButchering',l:'停用屠宰'},
                    {k:'disableRenaming',l:'停用角色改名'},
                    {k:'disablePalRenaming',l:'停用帕鲁改名'},
                    {k:'RCONUsePacketIdFix',l:'修正 RCON 封包 ID 问题'},
                  ]" :key="sw.k">
                    <div class="gcfg-rate-item" style="padding:4px 0">
                      <div class="gcfg-rate-label"><span>{{ sw.l }}</span><span v-if="sw.d" class="gcfg-rate-key">{{ sw.d }}</span></div>
                      <n-switch v-model:value="pd[sw.k]" size="small"/>
                    </div>
                  </n-gi>
                </n-grid>
                <n-grid :cols="2" :x-gap="12" :y-gap="6" class="mt-3">
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>钻油平台宝箱锁定时间(s) (OilrigGoalBoxLocktime)</span></div>
                      <n-input-number v-model:value="pd.OilrigGoalBoxLocktime" :min="0" :max="3600" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="gcfg-rate-item">
                      <div class="gcfg-rate-label"><span>RCON 连接超时(s) (RCONTimeout)</span></div>
                      <n-input-number v-model:value="pd.RCONTimeout" :min="1" :max="60" :step="0.5" size="small" style="width:140px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

            </n-spin>
          </div>
          <div v-else class="gcfg-pane">
            <n-input v-model:value="rawContent.paldefender" type="textarea" :rows="30" style="font-family:monospace;font-size:12px"/>
          </div>
        </n-scrollbar>
      </n-tab-pane>

    </n-tabs>

    <template #footer>
      <n-flex justify="space-between" align="center">
        <n-text depth="3" style="font-size:11px;max-width:460px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">
          {{ filePaths[activeTab] || "路径未知" }}
        </n-text>
        <n-flex gap="8">
          <n-button size="small" @click="rawMode[activeTab] = !rawMode[activeTab]">
            {{ rawMode[activeTab] ? "可视化编辑" : "编辑原始文件" }}
          </n-button>
          <n-button size="small" @click="loadTab(activeTab)" :loading="loading">刷新</n-button>
          <n-button size="small" type="primary" @click="saveTab(activeTab)" :loading="saving">保存</n-button>
        </n-flex>
      </n-flex>
    </template>
  </n-modal>
</template>

<style scoped lang="less">
.gcfg-pane { padding: 12px 16px; }
.gcfg-label { font-size: 13px; }
.gcfg-toggle-row {
  padding: 5px 0;
  border-bottom: 1px solid rgba(128,128,128,0.1);
}
.gcfg-rate-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 5px 0;
  border-bottom: 1px solid rgba(128,128,128,0.1);
}
.gcfg-rate-label {
  display: flex;
  flex-direction: column;
  gap: 1px;
  span:first-child { font-size: 13px; font-weight: 500; }
}
.gcfg-rate-key {
  font-size: 10px;
  opacity: 0.45;
  font-family: monospace;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
