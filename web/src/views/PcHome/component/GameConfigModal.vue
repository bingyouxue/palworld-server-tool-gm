<script setup>
import { ref, watch, computed } from "vue";
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
const filePaths = ref({ world: "", engine: "", paldefender: "" });
const rawMode = ref({ world: false, engine: false, paldefender: false });
const worldCategory = ref("server");
const wf = ref({
  Difficulty: "None", RandomizerType: "None", RandomizerSeed: "",
  bIsRandomizerPalLevelRandom: false,
  DayTimeSpeedRate: 1.0, NightTimeSpeedRate: 1.0, ExpRate: 1.0,
  PalCaptureRate: 1.0, PalSpawnNumRate: 1.0,
  PalDamageRateAttack: 1.0, PalDamageRateDefense: 1.0,
  PlayerDamageRateAttack: 1.0, PlayerDamageRateDefense: 1.0,
  PlayerStomachDecreaceRate: 1.0, PlayerStaminaDecreaceRate: 1.0,
  PlayerAutoHPRegeneRate: 1.0, PlayerAutoHpRegeneRateInSleep: 1.0,
  PalStomachDecreaceRate: 1.0, PalStaminaDecreaceRate: 1.0,
  PalAutoHPRegeneRate: 1.0, PalAutoHpRegeneRateInSleep: 1.0,
  BuildObjectHpRate: 1.0, BuildObjectDamageRate: 1.0,
  BuildObjectDeteriorationDamageRate: 1.0,
  CollectionDropRate: 1.0, CollectionObjectHpRate: 1.0,
  CollectionObjectRespawnSpeedRate: 1.0,
  EnemyDropItemRate: 1.0, DeathPenalty: "Item",
  bEnablePlayerToPlayerDamage: false, bEnableFriendlyFire: false,
  bEnableInvaderEnemy: true, bActiveUNKO: false,
  bEnableAimAssistPad: true, bEnableAimAssistKeyboard: false,
  DropItemMaxNum: 3000, PhysicsActiveDropItemMaxNum: -1,
  DropItemMaxNum_UNKO: 100,
  BaseCampMaxNum: 128, BaseCampWorkerMaxNum: 15,
  DropItemAliveMaxHours: 1.0,
  bAutoResetGuildNoOnlinePlayers: false,
  AutoResetGuildTimeNoOnlinePlayers: 72.0,
  GuildPlayerMaxNum: 20, BaseCampMaxNumInGuild: 4,
  PalEggDefaultHatchingTime: 1.0, WorkSpeedRate: 1.0, AutoSaveSpan: 30.0,
  bIsMultiplay: false, bIsPvP: false, bHardcore: false, bPalLost: false,
  bCharacterRecreateInHardcore: false,
  bCanPickupOtherGuildDeathPenaltyDrop: false,
  bEnableNonLoginPenalty: true, bEnableFastTravel: true,
  bIsStartLocationSelectByMap: false, bExistPlayerAfterLogout: false,
  bEnableDefenseOtherGuildPlayer: false,
  bInvisibleOtherGuildBaseCampAreaFX: false,
  bBuildAreaLimit: false, ItemWeightRate: 1.0,
  CoopPlayerMaxNum: 4, ServerPlayerMaxNum: 32,
  ServerName: "Default Palworld Server", ServerDescription: "",
  AdminPassword: "", ServerPassword: "",
  bAllowClientMod: true, PublicPort: 8211, PublicIP: "",
  RCONEnabled: false, RCONPort: 25575, Region: "", bUseAuth: true,
  BanListURL: "https://b.palworldgame.com/api/banlist.txt",
  RESTAPIEnabled: false, RESTAPIPort: 8212, bShowPlayerList: false,
  ChatPostLimitPerMinute: 30,
  CrossplayPlatforms: "(Steam,Xbox,PS5,Mac)",
  bIsUseBackupSaveData: true, LogFormatType: "Text",
  bIsShowJoinLeftMessage: true,
  SupplyDropSpan: 180, EnablePredatorBossPal: true, MaxBuildingLimitNum: 0,
  ServerReplicatePawnCullDistance: 15000.0,
  bAllowGlobalPalboxExport: true, bAllowGlobalPalboxImport: false,
  EquipmentDurabilityDamageRate: 1.0,
  ItemContainerForceMarkDirtyInterval: 1.0,
  PlayerDataPalStorageUpdateCheckTickInterval: 1.0,
  ItemCorruptionMultiplier: 1.0, MonsterFarmActionSpeedRate: 1.0,
  DenyTechnologyList: "", GuildRejoinCooldownMinutes: 0,
  AutoTransferMasterCheckIntervalSeconds: 3600.0,
  AutoTransferMasterThresholdDays: 14,
  MaxGuildsPerFrame: 10, BlockRespawnTime: 5.0,
  RespawnPenaltyDurationThreshold: 0.0, RespawnPenaltyTimeScale: 2.0,
  bDisplayPvPItemNumOnWorldMap_BaseCamp: false,
  bDisplayPvPItemNumOnWorldMap_Player: false,
  AdditionalDropItemWhenPlayerKillingInPvPMode: "PlayerDropItem",
  AdditionalDropItemNumWhenPlayerKillingInPvPMode: 1,
  bAdditionalDropItemWhenPlayerKillingInPvPMode: false,
  bEnableVoiceChat: false, VoiceChatMaxVolumeDistance: 3000.0,
  VoiceChatZeroVolumeDistance: 15000.0,
  bAllowEnhanceStat_Health: true, bAllowEnhanceStat_Attack: true,
  bAllowEnhanceStat_Stamina: true, bAllowEnhanceStat_Weight: true,
  bAllowEnhanceStat_WorkSpeed: true,
  bEnableBuildingPlayerUIdDisplay: false,
  BuildingNameDisplayCacheTTLSeconds: 60,
});
const ef = ref({
  NetServerMaxTickRate: 0, MaxClientRate: 0, MaxInternetClientRate: 0,
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
  shouldKickCheaters: false, shouldBanCheaters: false,
  shouldIPBanCheaters: false,
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
  logChat: true, logRCON: false, logPlayerLogins: true,
  logPlayerDeaths: true, logPlayerBuildings: false,
  logPlayerSummons: false, logPlayerCaptures: true,
  logCraftings: false, logTechUnlocks: false,
  logPlayerUID: true, logPlayerIP: false, logNetworking: false,
  exitServerOnStartupFailure: true, disableButchering: false,
  disableRenaming: false, disablePalRenaming: false,
  OilrigGoalBoxLocktime: 300, RCONTimeout: 31, RCONUsePacketIdFix: true,
});
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
  if (Object.keys(kv).length === 0) return;
  const f = wf.value;
  Object.keys(f).forEach(k => {
    if (kv[k] === undefined) return;
    const v = kv[k];
    if (typeof f[k] === "boolean") f[k] = v.toLowerCase() === "true";
    else if (typeof f[k] === "number") {
      const n = parseFloat(v);
      if (!isNaN(n)) f[k] = n;
    } else f[k] = v.replace(/^"|"$/g, "");
  });
}

function worldToIni() {
  const f = wf.value;
  const pairs = Object.entries(f).map(([k, v]) => {
    if (typeof v === "boolean") return `${k}=${v ? "True" : "False"}`;
    if (typeof v === "number") return `${k}=${v}`;
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
    else if (typeof f[key] === "number") {
      const n = parseFloat(val);
      if (!isNaN(n)) f[key] = n;
    }
  }
}

function engineToIni() {
  const f = ef.value;

  // 受控 key 定义：section -> { key -> 当前值或 null 表示删除 }
  // key 为 null 表示该行应被移除（值回到默认、不写入）
  const controlled = {
    "[/Script/OnlineSubsystemUtils.IpNetDriver]": {
      "NetServerMaxTickRate": f.NetServerMaxTickRate > 0 ? String(f.NetServerMaxTickRate) : null,
    },
    "[/Script/Engine.GameNetworkManager]": {
      "MaxClientRate":           f.MaxClientRate > 0 ? String(f.MaxClientRate) : null,
      "MaxInternetClientRate":   f.MaxInternetClientRate > 0 ? String(f.MaxInternetClientRate) : null,
      "ConnectionTimeout":       f.ConnectionTimeout > 0 ? String(f.ConnectionTimeout) : null,
      "InitialConnectTimeout":   f.InitialConnectTimeout > 0 ? String(f.InitialConnectTimeout) : null,
    },
    "[/Script/Engine.Engine]": {
      "bUseFixedFrameRate": f.bUseFixedFrameRate ? "True" : null,
      "FixedFrameRate":     f.bUseFixedFrameRate ? String(f.FixedFrameRate) : null,
      "bSmoothFrameRate":   f.bSmoothFrameRate ? "True" : null,
    },
    "[ConsoleVariables]": {
      "gc.TimeBetweenPurgingPendingKillObjects": f.gcTimeBetweenPurging > 0 ? String(f.gcTimeBetweenPurging) : null,
    },
    "[URL]": {
      "useperfthreads":            f.useperfthreads ? "" : null,
      "NoAsyncLoadingThread":      f.NoAsyncLoadingThread ? "" : null,
      "UseMultithreadForDS":       f.UseMultithreadForDS ? "" : null,
      "NumberOfWorkerThreadsServer": f.NumberOfWorkerThreadsServer > 0 ? `NumberOfWorkerThreadsServer=${f.NumberOfWorkerThreadsServer}` : null,
    },
  };

  // 以原始文件内容为基础进行 patch
  const base = rawContent.value['engine'] || '';
  const srcLines = base.split(/\r?\n/);
  const outLines = [];
  let curSection = '';
  // 记录哪些 key 已经被处理（已存在于原文件并已更新/删除）
  const handled = {};  // section -> Set<key>

  for (let i = 0; i < srcLines.length; i++) {
    const raw = srcLines[i];
    const t = raw.trim();

    // section 标题
    if (t.startsWith('[') && t.endsWith(']')) {
      // 在离开上一个 section 前，追加该 section 中尚未出现的 key（新增）
      if (curSection && controlled[curSection]) {
        const done = handled[curSection] || new Set();
        for (const [k, v] of Object.entries(controlled[curSection])) {
          if (!done.has(k) && v !== null) {
            // [URL] section 中 value=="" 表示无等号的开关行，特殊处理
            outLines.push(curSection === '[URL]' && v === '' ? k : `${k}=${v}`);
          }
        }
      }
      curSection = t;
      outLines.push(raw);
      if (!handled[curSection]) handled[curSection] = new Set();
      continue;
    }

    // 空行或注释原样保留
    if (!t || t.startsWith(';') || t.startsWith('#')) {
      outLines.push(raw);
      continue;
    }

    // 普通 key=value 或裸 flag 行
    if (curSection && controlled[curSection]) {
      const eqIdx = t.indexOf('=');
      // 裸 flag（如 useperfthreads）
      const key = eqIdx >= 0 ? t.slice(0, eqIdx).trim() : t;

      if (key in controlled[curSection]) {
        handled[curSection].add(key);
        const newVal = controlled[curSection][key];
        if (newVal === null) {
          // 删除：跳过此行
          continue;
        } else if (curSection === '[URL]' && newVal === '') {
          outLines.push(key);  // 裸 flag 保留
        } else if (eqIdx >= 0) {
          outLines.push(`${key}=${newVal}`);
        } else {
          // 原文件是裸 flag，但现在有值（不应发生，保险处理）
          outLines.push(`${key}=${newVal}`);
        }
        continue;
      }
    }

    outLines.push(raw);
  }

  // 文件结束后，处理原文件中完全不存在的 section（整体追加）
  if (curSection && controlled[curSection]) {
    const done = handled[curSection] || new Set();
    for (const [k, v] of Object.entries(controlled[curSection])) {
      if (!done.has(k) && v !== null) {
        outLines.push(curSection === '[URL]' && v === '' ? k : `${k}=${v}`);
      }
    }
  }
  // 追加原文件中根本不存在的 section
  for (const [sec, kvs] of Object.entries(controlled)) {
    if (handled[sec]) continue;
    const toAdd = Object.entries(kvs).filter(([, v]) => v !== null);
    if (!toAdd.length) continue;
    outLines.push('', sec);
    for (const [k, v] of toAdd) {
      outLines.push(sec === '[URL]' && v === '' ? k : `${k}=${v}`);
    }
  }

  return outLines.join('\n');
}
async function loadTab(tab) {
  loading.value = true;
  try {
    const { data } = await api.getGameConfig(tab);
    const d = data.value;
    if (d) {
      rawContent.value[tab] = d.content || "";
      filePaths.value[tab] = d.path || "";
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
    : tab === "world" ? worldToIni()
    : tab === "engine" ? engineToIni()
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
const WORLD_CATEGORIES = [
  { key: "server", label: "服务器" },
  { key: "world", label: "世界" },
  { key: "pal", label: "帕鲁" },
  { key: "player", label: "玩家" },
  { key: "guild", label: "公会" },
  { key: "build", label: "建筑" },
  { key: "drop", label: "物品掉落" },
];

const CAT_SERVER = {
  strings: [
    { k: "ServerName",        l: "服务器名称",     d: "服务器列表中显示的名称",           ph: "" },
    { k: "ServerDescription", l: "服务器描述",     d: "详细描述，显示在服务器信息中",       ph: "" },
    { k: "ServerPassword",    l: "服务器密码",     d: "加入服务器所需密码，留空则公开",     ph: "", secret: true },
    { k: "AdminPassword",     l: "管理员密码",     d: "RCON 及管理员命令使用的密码",       ph: "", secret: true },
    { k: "PublicIP",          l: "公开 IP",       d: "对外公开的 IP 地址，留空自动检测",   ph: "留空自动检测" },
    { k: "Region",            l: "服务器地区",     d: "服务器所在区域标签，留空不限制",     ph: "留空" },
    { k: "CrossplayPlatforms",l: "跨平台连接",    d: "允许连接的平台列表",               ph: "(Steam,Xbox,PS5,Mac)" },
    { k: "BanListURL",        l: "封禁名单 URL",  d: "远程封禁列表地址，定期自动拉取",     ph: "" },
  ],
  ints: [
    { k: "ServerPlayerMaxNum",               l: "最大玩家数",           d: "服务器允许同时在线的玩家上限",             mn: 1,    mx: 99 },
    { k: "CoopPlayerMaxNum",                 l: "合作玩家数上限",         d: "单个公会/小队的合作人数上限",              mn: 1,    mx: 8 },
    { k: "PublicPort",                       l: "游戏端口",             d: "游戏连接端口，默认 8211",                 mn: 1024, mx: 65535 },
    { k: "RESTAPIPort",                      l: "REST API 端口",        d: "HTTP REST API 监听端口，默认 8212",       mn: 1024, mx: 65535 },
    { k: "RCONPort",                         l: "RCON 端口",           d: "远程控制台 RCON 监听端口，默认 25575",     mn: 1024, mx: 65535 },
    { k: "ChatPostLimitPerMinute",           l: "每分钟聊天上限",         d: "单个玩家每分钟最多发送的聊天条数",           mn: 1,    mx: 120 },
    { k: "MaxGuildsPerFrame",               l: "每帧处理公会数",          d: "每帧最多处理的公会数，影响公会同步性能",      mn: 1,    mx: 100 },
    { k: "AutoTransferMasterThresholdDays", l: "自动转移天数门槛",        d: "公会长离线超过此天数后自动转移领导权",        mn: 0,    mx: 365 },
  ],
  floats: [
    { k: "AutoSaveSpan",                             l: "自动存档间隔(秒)",      d: "自动保存游戏数据的间隔时间",                    mn: 10, mx: 600,   st: 5 },
    { k: "VoiceChatMaxVolumeDistance",               l: "语音最大距离",          d: "能听到语音聊天的最大距离",                      mn: 0,  mx: 50000, st: 100 },
    { k: "VoiceChatZeroVolumeDistance",              l: "语音零音量距离",         d: "语音音量降为零的距离",                          mn: 0,  mx: 50000, st: 100 },
    { k: "AutoTransferMasterCheckIntervalSeconds",   l: "自动转移检查间隔(秒)",   d: "检查公会长是否需要自动转移的时间间隔",             mn: 60, mx: 86400, st: 1 },
    { k: "ItemContainerForceMarkDirtyInterval",      l: "物品容器同步间隔",       d: "强制标记物品容器为脏数据的间隔，影响同步频率",      mn: 0,  mx: 60,    st: 0.1 },
    { k: "PlayerDataPalStorageUpdateCheckTickInterval", l: "帕鲁箱更新检查间隔", d: "帕鲁存储箱更新检查的 Tick 间隔",                mn: 0,  mx: 60,    st: 0.1 },
  ],
  toggles: [
    { k: "bIsMultiplay",           l: "开启多人游戏",      d: "允许多人联机，关闭则只能单人游戏" },
    { k: "bShowPlayerList",        l: "显示玩家列表",      d: "在服务器信息中公开在线玩家列表" },
    { k: "bIsShowJoinLeftMessage", l: "显示加入/离开消息",  d: "玩家加入或离开时在聊天中广播消息" },
    { k: "RESTAPIEnabled",         l: "启用 REST API",    d: "开启 HTTP REST API，供第三方工具查询服务器数据" },
    { k: "RCONEnabled",            l: "启用 RCON",        d: "开启远程控制台，允许通过 RCON 执行命令" },
    { k: "bUseAuth",               l: "官方账号验证",      d: "要求玩家使用官方平台账号登录" },
    { k: "bAllowClientMod",        l: "允许客户端模组",    d: "允许客户端安装模组并连接服务器" },
    { k: "bIsUseBackupSaveData",   l: "启用存档备份",      d: "自动备份存档数据，异常时可恢复" },
    { k: "bEnableVoiceChat",       l: "启用语音聊天",      d: "开启内置语音聊天功能" },
  ],
  enums: [
    { k: "LogFormatType", l: "日志格式", d: "服务器日志的输出格式", opts: [{l:"文字",v:"Text"},{l:"JSON",v:"Json"}] },
  ],
};

const CAT_WORLD = {
  floats: [
    { k: "DayTimeSpeedRate",                 l: "白天流逝速度",   d: "白天时间流逝的倍率，数值越大白天越短",   mn: 0.1, mx: 5,  st: 0.1 },
    { k: "NightTimeSpeedRate",               l: "夜晚流逝速度",   d: "夜晚时间流逝的倍率，数值越大夜晚越短",   mn: 0.1, mx: 5,  st: 0.1 },
    { k: "CollectionObjectHpRate",           l: "采集物耐久倍率", d: "树木、矿石等可采集物的耐久倍率",          mn: 0.5, mx: 20, st: 0.1 },
    { k: "CollectionObjectRespawnSpeedRate", l: "采集物重生速度", d: "采集物重新生成的速度倍率",               mn: 0.5, mx: 20, st: 0.1 },
  ],
  toggles: [
    { k: "bEnableInvaderEnemy",           l: "启用突袭事件",      d: "开启敌方势力不定期突袭玩家据点的事件" },
    { k: "bEnableAimAssistPad",           l: "手柄辅助瞄准",      d: "使用手柄/控制器时启用自动辅助瞄准" },
    { k: "bEnableAimAssistKeyboard",      l: "键鼠辅助瞄准",      d: "使用键盘鼠标时启用自动辅助瞄准" },
    { k: "bHardcore",                     l: "硬核模式",          d: "开启后死亡会删除角色，难度大幅提升" },
    { k: "bCharacterRecreateInHardcore",  l: "硬核可重建角色",    d: "硬核模式下角色死亡后是否允许重新创建" },
    { k: "bIsRandomizerPalLevelRandom",   l: "随机化帕鲁等级",    d: "随机化模式下帕鲁等级是否随机分配" },
  ],
  enums: [
    { k: "Difficulty",     l: "游戏难度",   d: "整体游戏难度预设", opts: [{l:"无",v:"None"},{l:"休闲",v:"Casual"},{l:"普通",v:"Normal"},{l:"困难",v:"Hard"}] },
    { k: "RandomizerType", l: "随机化模式", d: "帕鲁刷新的随机化范围",  opts: [{l:"无",v:"None"},{l:"区域",v:"Region"},{l:"全部",v:"All"}] },
  ],
  strings: [
    { k: "RandomizerSeed", l: "随机化种子", d: "固定随机化种子以复现相同结果，留空则每次随机", ph: "留空=随机" },
  ],
  ints: [],
};
const CAT_PAL = {
  floats: [
    { k: "PalCaptureRate",              l: "捕获率倍率",       d: "帕鲁球捕获成功率的倍率",                       mn: 0.5, mx: 20, st: 0.1 },
    { k: "PalSpawnNumRate",             l: "帕鲁出现数量倍率", d: "野外帕鲁刷新数量的倍率",                        mn: 0.5, mx: 20, st: 0.1 },
    { k: "PalDamageRateAttack",         l: "帕鲁攻击倍率",     d: "帕鲁造成伤害的倍率",                           mn: 0.1, mx: 20, st: 0.1 },
    { k: "PalDamageRateDefense",        l: "帕鲁防御倍率",     d: "帕鲁承受伤害的倍率（越高越耐打）",               mn: 0.1, mx: 20, st: 0.1 },
    { k: "PalStomachDecreaceRate",      l: "帕鲁饱食度消耗",   d: "帕鲁饱食度下降速度的倍率，越低越不易饿",          mn: 0.1, mx: 20, st: 0.1 },
    { k: "PalStaminaDecreaceRate",      l: "帕鲁体力消耗",     d: "帕鲁行动体力消耗的倍率",                       mn: 0.1, mx: 20, st: 0.1 },
    { k: "PalAutoHPRegeneRate",         l: "帕鲁自动恢复HP",   d: "帕鲁在野外自动回血的速度倍率",                  mn: 0.1, mx: 20, st: 0.1 },
    { k: "PalAutoHpRegeneRateInSleep",  l: "帕鲁睡眠恢复HP",   d: "帕鲁在据点睡眠时回血的速度倍率",                mn: 0.1, mx: 20, st: 0.1 },
    { k: "PalEggDefaultHatchingTime",   l: "蛋孵化时间(小时)", d: "普通品质帕鲁蛋完成孵化所需小时数",               mn: 0,   mx: 240, st: 1 },
    { k: "WorkSpeedRate",               l: "工作速度倍率",     d: "帕鲁在据点进行各类工作的速度倍率",               mn: 0.1, mx: 20, st: 0.1 },
    { k: "MonsterFarmActionSpeedRate",  l: "牧场工作速度",     d: "帕鲁牧场（MonsterFarm）特定动作的速度倍率",     mn: 0.1, mx: 20, st: 0.1 },
  ],
  toggles: [
    { k: "bPalLost",                   l: "帕鲁死亡永久消失",    d: "开启后帕鲁死亡将从队伍中永久移除" },
    { k: "bAllowGlobalPalboxExport",   l: "允许全域帕鲁箱导出",  d: "允许玩家将帕鲁导出到全域存储箱" },
    { k: "bAllowGlobalPalboxImport",   l: "允许全域帕鲁箱导入",  d: "允许玩家从全域存储箱导入帕鲁" },
    { k: "EnablePredatorBossPal",      l: "启用掠食者首领",      d: "在世界中生成拥有特殊AI的掠食者首领帕鲁" },
  ],
  ints: [],
  strings: [],
  enums: [],
};

const CAT_PLAYER = {
  floats: [
    { k: "ExpRate",                         l: "经验值倍率",          d: "玩家获得经验值的倍率",                         mn: 0.1, mx: 20,   st: 0.1 },
    { k: "PlayerDamageRateAttack",          l: "玩家攻击倍率",         d: "玩家造成伤害的倍率",                           mn: 0.1, mx: 20,   st: 0.1 },
    { k: "PlayerDamageRateDefense",         l: "玩家防御倍率",         d: "玩家承受伤害的倍率（越高越耐打）",               mn: 0.1, mx: 20,   st: 0.1 },
    { k: "PlayerStomachDecreaceRate",       l: "玩家饱食度消耗",       d: "玩家饱食度下降速度的倍率",                     mn: 0.1, mx: 5,    st: 0.1 },
    { k: "PlayerStaminaDecreaceRate",       l: "玩家体力消耗",         d: "玩家行动体力消耗的倍率",                       mn: 0.1, mx: 5,    st: 0.1 },
    { k: "PlayerAutoHPRegeneRate",          l: "玩家自动恢复HP",       d: "玩家在野外自动回血的速度倍率",                  mn: 0.1, mx: 5,    st: 0.1 },
    { k: "PlayerAutoHpRegeneRateInSleep",   l: "玩家睡眠恢复HP",       d: "玩家在基地睡眠时回血的速度倍率",                mn: 0.1, mx: 5,    st: 0.1 },
    { k: "ItemWeightRate",                  l: "物品重量倍率",         d: "所有物品重量的倍率，影响玩家负重上限利用率",     mn: 0.1, mx: 20,   st: 0.1 },
    { k: "EquipmentDurabilityDamageRate",   l: "装备耐久损耗倍率",     d: "装备使用时耐久度下降速度的倍率",                mn: 0.1, mx: 20,   st: 0.1 },
    { k: "BlockRespawnTime",                l: "重生封锁时间(秒)",     d: "玩家死亡后不可立即复活的等待时间",              mn: 0,   mx: 3600, st: 0.5 },
    { k: "RespawnPenaltyDurationThreshold", l: "重生惩罚门槛(秒)",     d: "触发重生惩罚所需的死亡后最短等待时间",          mn: 0,   mx: 3600, st: 0.5 },
    { k: "RespawnPenaltyTimeScale",         l: "重生惩罚时间倍率",     d: "重生惩罚时长的缩放倍率",                       mn: 0,   mx: 10,   st: 0.1 },
  ],
  toggles: [
    { k: "bEnablePlayerToPlayerDamage",                  l: "玩家间伤害",          d: "允许玩家对其他玩家造成伤害" },
    { k: "bEnableFriendlyFire",                          l: "友军伤害",            d: "允许对同公会成员造成伤害" },
    { k: "bIsPvP",                                       l: "开启 PvP",            d: "启用玩家对战模式" },
    { k: "bEnableFastTravel",                            l: "启用快速旅行",         d: "允许玩家使用快速旅行传送点" },
    { k: "bIsStartLocationSelectByMap",                  l: "地图选择出生点",       d: "玩家可通过地图选择出生位置" },
    { k: "bExistPlayerAfterLogout",                      l: "下线后角色留存",       d: "玩家下线后角色实体仍留在世界中" },
    { k: "bEnableNonLoginPenalty",                       l: "启用离线惩罚",         d: "长时间未登录的玩家将受到惩罚" },
    { k: "bAllowEnhanceStat_Health",                     l: "允许强化生命值",       d: "允许玩家提升生命值属性点" },
    { k: "bAllowEnhanceStat_Attack",                     l: "允许强化攻击力",       d: "允许玩家提升攻击力属性点" },
    { k: "bAllowEnhanceStat_Stamina",                    l: "允许强化耐力",         d: "允许玩家提升耐力属性点" },
    { k: "bAllowEnhanceStat_Weight",                     l: "允许强化负重",         d: "允许玩家提升负重属性点" },
    { k: "bAllowEnhanceStat_WorkSpeed",                  l: "允许强化工作速度",     d: "允许玩家提升工作速度属性点" },
    { k: "bAdditionalDropItemWhenPlayerKillingInPvPMode",l: "PvP击杀额外掉落",     d: "PvP 击杀时额外掉落道具" },
    { k: "bDisplayPvPItemNumOnWorldMap_Player",          l: "地图显示PvP掉落数",   d: "在世界地图上显示玩家 PvP 掉落物数量" },
  ],
  ints: [
    { k: "AdditionalDropItemNumWhenPlayerKillingInPvPMode", l: "PvP击杀额外掉落数量", d: "PvP 击杀时额外掉落的道具数量", mn: 0, mx: 9999 },
  ],
  enums: [
    { k: "DeathPenalty",                              l: "死亡惩罚",        d: "玩家死亡时的惩罚类型",      opts: [{l:"无",v:"None"},{l:"仅道具",v:"Item"},{l:"道具+装备",v:"ItemAndEquipment"},{l:"全部",v:"All"}] },
    { k: "AdditionalDropItemWhenPlayerKillingInPvPMode", l: "PvP掉落类型", d: "PvP 击杀时额外掉落的内容", opts: [{l:"无",v:"None"},{l:"玩家掉落物",v:"PlayerDropItem"},{l:"全部物品",v:"AllItems"}] },
  ],
  strings: [],
};
const CAT_GUILD = {
  ints: [
    { k: "GuildPlayerMaxNum",              l: "公会人数上限",          d: "单个公会允许的最大成员数量",              mn: 1, mx: 100 },
    { k: "BaseCampMaxNum",                 l: "世界据点总上限",        d: "整个服务器允许存在的据点总数量",           mn: 1, mx: 1024 },
    { k: "BaseCampMaxNumInGuild",          l: "每公会据点上限",        d: "单个公会允许建造的据点数量上限",           mn: 1, mx: 10 },
    { k: "BaseCampWorkerMaxNum",           l: "据点工作帕鲁上限",      d: "单个据点可分配的工作帕鲁数量上限",        mn: 1, mx: 50 },
    { k: "GuildRejoinCooldownMinutes",     l: "重新加入公会冷却(分)",  d: "离开公会后重新加入其他公会的等待时间",    mn: 0, mx: 10080 },
  ],
  floats: [
    { k: "AutoResetGuildTimeNoOnlinePlayers", l: "公会离线清除时间(小时)", d: "公会全员离线超过此时间后自动解散公会及据点", mn: 1, mx: 168, st: 1 },
  ],
  toggles: [
    { k: "bAutoResetGuildNoOnlinePlayers",        l: "自动清除离线公会",    d: "达到离线时间上限后自动解散长期无人的公会" },
    { k: "bEnableDefenseOtherGuildPlayer",        l: "防御其他公会玩家",    d: "允许玩家在他人公会范围内进行防御" },
    { k: "bCanPickupOtherGuildDeathPenaltyDrop",  l: "可拾取他人死亡掉落",  d: "允许拾取其他公会成员死亡后的掉落物" },
    { k: "bInvisibleOtherGuildBaseCampAreaFX",    l: "隐藏其他公会特效",    d: "隐藏其他公会据点的视觉特效，优化性能" },
    { k: "bDisplayPvPItemNumOnWorldMap_BaseCamp", l: "地图显示据点PvP掉落", d: "在世界地图上显示据点 PvP 掉落物数量" },
  ],
  strings: [],
  enums: [],
};

const CAT_BUILD = {
  floats: [
    { k: "BuildObjectDamageRate",              l: "建筑受损倍率",     d: "建筑物受到攻击时伤害的倍率",                     mn: 0.1, mx: 20,    st: 0.1 },
    { k: "BuildObjectDeteriorationDamageRate", l: "建筑劣化速度倍率", d: "建筑物随时间自然劣化损坏的速度倍率",              mn: 0,   mx: 20,    st: 0.1 },
    { k: "BuildObjectHpRate",                  l: "建筑血量倍率",     d: "建筑物最大耐久度/血量的倍率",                     mn: 0.1, mx: 20,    st: 0.1 },
    { k: "ServerReplicatePawnCullDistance",    l: "角色同步距离",     d: "服务器同步玩家/帕鲁的最大距离，超过后停止同步",   mn: 5000, mx: 15000, st: 100 },
  ],
  ints: [
    { k: "MaxBuildingLimitNum",              l: "建筑数量上限",      d: "服务器全局建筑物数量上限，0 表示无限制",        mn: 0, mx: 10000 },
    { k: "BuildingNameDisplayCacheTTLSeconds", l: "建筑名称缓存(秒)", d: "建筑物所属玩家名称的客户端缓存有效时间",       mn: 0, mx: 600 },
  ],
  toggles: [
    { k: "bBuildAreaLimit",                  l: "限制建筑范围",      d: "限制只能在特定区域内建造建筑物" },
    { k: "bEnableBuildingPlayerUIdDisplay",  l: "显示建筑所属玩家",  d: "显示建筑物归属的玩家 ID 信息" },
  ],
  strings: [],
  enums: [],
};

const CAT_DROP = {
  floats: [
    { k: "DropItemAliveMaxHours",      l: "掉落物存活时间(小时)", d: "掉落物在地面上存在的最长时间",            mn: 0,   mx: 24, st: 0.5 },
    { k: "CollectionDropRate",         l: "采集掉落倍率",         d: "采集树木/矿石时获得材料的数量倍率",       mn: 0.5, mx: 20, st: 0.1 },
    { k: "EnemyDropItemRate",          l: "敌人掉落倍率",         d: "击败敌方帕鲁或NPC时掉落物品的数量倍率",   mn: 0.5, mx: 20, st: 0.1 },
    { k: "ItemCorruptionMultiplier",   l: "食物腐败速度倍率",     d: "食物腐败变质的速度倍率",                 mn: 0.1, mx: 20, st: 0.1 },
  ],
  ints: [
    { k: "DropItemMaxNum",                l: "掉落物上限",           d: "服务器全局掉落物数量上限",                           mn: 0,  mx: 5000 },
    { k: "DropItemMaxNum_UNKO",           l: "粪便掉落物上限",       d: "服务器全局粪便类掉落物数量上限",                     mn: 0,  mx: 5000 },
    { k: "SupplyDropSpan",               l: "空投补给间隔(分)",     d: "空投补给箱刷新的时间间隔（分钟）",                   mn: 30, mx: 1440 },
    { k: "PhysicsActiveDropItemMaxNum",  l: "物理掉落物上限",       d: "激活物理模拟的掉落物数量上限，-1 表示无限",          mn: -1, mx: 10000 },
  ],
  toggles: [
    { k: "bActiveUNKO", l: "启用粪便掉落物", d: "开启帕鲁会排泄产生粪便掉落物的功能" },
  ],
  strings: [
    { k: "DenyTechnologyList", l: "禁止的科技清单", d: "禁止玩家研究的科技 ID 列表，多个用逗号分隔", ph: "" },
  ],
  enums: [],
};

const categoryConfigs = {
  server: CAT_SERVER,
  world: CAT_WORLD,
  pal: CAT_PAL,
  player: CAT_PLAYER,
  guild: CAT_GUILD,
  build: CAT_BUILD,
  drop: CAT_DROP,
};

const currentCatConfig = computed(() => categoryConfigs[worldCategory.value] || {});
</script>
<template>
  <n-modal :show="show" preset="card" title="配置设置"
    style="width:96%;max-width:900px;max-height:92vh"
    content-style="padding:0;overflow:hidden"
    :bordered="false"
    @update:show="emit('update:show',$event)">

    <n-tabs v-model:value="activeTab" type="line" animated justify-content="space-evenly">

      <!-- 世界设置 -->
      <n-tab-pane name="world" tab="世界设置">
        <n-scrollbar style="max-height:calc(92vh - 155px)">
          <div class="gcfg-pane" v-if="!rawMode.world">
            <n-spin :show="loading">
              <!-- 分类按钮 -->
              <div class="gcfg-cat-bar">
                <button v-for="cat in WORLD_CATEGORIES" :key="cat.key"
                  :class="['gcfg-cat-btn', worldCategory === cat.key && 'active']"
                  @click="worldCategory = cat.key">{{ cat.label }}</button>
              </div>

              <!-- 枚举选择 -->
              <template v-if="currentCatConfig.enums && currentCatConfig.enums.length">
                <div class="gcfg-section-title">选项</div>
                <n-grid :cols="2" :x-gap="12" :y-gap="8">
                  <n-gi v-for="field in currentCatConfig.enums" :key="field.k">
                    <div class="gcfg-field-wrap">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ field.l }}</span>
                        <span class="gcfg-field-key">{{ field.k }}</span>
                      </div>
                      <p v-if="field.d" class="gcfg-field-desc">{{ field.d }}</p>
                      <n-select v-model:value="wf[field.k]" size="small"
                        :options="field.opts.map(o=>({label:o.l,value:o.v}))"/>
                    </div>
                  </n-gi>
                </n-grid>
              </template>

              <!-- 字符串 -->
              <template v-if="currentCatConfig.strings && currentCatConfig.strings.length">
                <div class="gcfg-section-title">文字设置</div>
                <n-grid :cols="2" :x-gap="12" :y-gap="8">
                  <n-gi v-for="field in currentCatConfig.strings" :key="field.k">
                    <div class="gcfg-field-wrap">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ field.l }}</span>
                        <span class="gcfg-field-key">{{ field.k }}</span>
                      </div>
                      <p v-if="field.d" class="gcfg-field-desc">{{ field.d }}</p>
                      <n-input v-model:value="wf[field.k]" size="small"
                        :placeholder="field.ph"
                        :type="field.secret ? 'password' : 'text'"
                        :show-password-on="field.secret ? 'click' : undefined"/>
                    </div>
                  </n-gi>
                </n-grid>
              </template>

              <!-- 整数 -->
              <template v-if="currentCatConfig.ints && currentCatConfig.ints.length">
                <div class="gcfg-section-title">数值设置</div>
                <n-grid :cols="2" :x-gap="12" :y-gap="6">
                  <n-gi v-for="field in currentCatConfig.ints" :key="field.k">
                    <div class="gcfg-slider-item">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ field.l }}</span>
                        <span class="gcfg-field-key">{{ field.k }}</span>
                      </div>
                      <p v-if="field.d" class="gcfg-field-desc">{{ field.d }}</p>
                      <div class="gcfg-slider-header">
                        <n-slider v-model:value="wf[field.k]" :min="field.mn" :max="field.mx" :step="1" style="flex:1"/>
                        <n-input-number v-model:value="wf[field.k]"
                          :min="field.mn" :max="field.mx" size="small" style="width:100px"/>
                      </div>
                    </div>
                  </n-gi>
                </n-grid>
              </template>

              <!-- 浮点数/倍率 -->
              <template v-if="currentCatConfig.floats && currentCatConfig.floats.length">
                <div class="gcfg-section-title">倍率 / 数值</div>
                <n-grid :cols="2" :x-gap="12" :y-gap="6">
                  <n-gi v-for="field in currentCatConfig.floats" :key="field.k">
                    <div class="gcfg-slider-item">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ field.l }}</span>
                        <span class="gcfg-field-key">{{ field.k }}</span>
                      </div>
                      <p v-if="field.d" class="gcfg-field-desc">{{ field.d }}</p>
                      <div class="gcfg-slider-header">
                        <n-slider v-model:value="wf[field.k]" :min="field.mn" :max="field.mx" :step="field.st" style="flex:1"/>
                        <n-input-number v-model:value="wf[field.k]"
                          :min="field.mn" :max="field.mx" :step="field.st"
                          size="small" style="width:100px"/>
                      </div>
                    </div>
                  </n-gi>
                </n-grid>
              </template>

              <!-- 开关 -->
              <template v-if="currentCatConfig.toggles && currentCatConfig.toggles.length">
                <div class="gcfg-section-title">开关</div>
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in currentCatConfig.toggles" :key="sw.k">
                    <div class="gcfg-toggle-row">
                      <div class="gcfg-toggle-info">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                        <p v-if="sw.d" class="gcfg-field-desc" style="margin:0">{{ sw.d }}</p>
                      </div>
                      <n-switch v-model:value="wf[sw.k]" size="small"/>
                    </div>
                  </n-gi>
                </n-grid>
              </template>

            </n-spin>
          </div>
          <div v-else class="gcfg-pane">
            <n-input v-model:value="rawContent.world" type="textarea" :rows="30"
              style="font-family:monospace;font-size:12px"/>
          </div>
        </n-scrollbar>
      </n-tab-pane>

      <!-- 引擎调优 -->
      <n-tab-pane name="engine" tab="引擎调优">
        <n-scrollbar style="max-height:calc(92vh - 155px)">
          <div class="gcfg-pane" v-if="!rawMode.engine">
            <n-spin :show="loading">
              <n-card size="small" class="mb-3" title="性能预设">
                <n-flex gap="8" wrap>
                  <n-button size="small" @click="Object.assign(ef,{NetServerMaxTickRate:0,MaxClientRate:0,MaxInternetClientRate:0,ConnectionTimeout:0,InitialConnectTimeout:0,bUseFixedFrameRate:false,bSmoothFrameRate:false,gcTimeBetweenPurging:0,useperfthreads:false,NoAsyncLoadingThread:false,UseMultithreadForDS:false,NumberOfWorkerThreadsServer:0})">游戏默认</n-button>
                  <n-button size="small" type="primary" @click="Object.assign(ef,{NetServerMaxTickRate:60,MaxClientRate:104857600,MaxInternetClientRate:104857600,ConnectionTimeout:120,InitialConnectTimeout:120,bUseFixedFrameRate:false,bSmoothFrameRate:false,gcTimeBetweenPurging:0,useperfthreads:false,NoAsyncLoadingThread:false,UseMultithreadForDS:false,NumberOfWorkerThreadsServer:0})">平衡</n-button>
                  <n-button size="small" type="warning" @click="Object.assign(ef,{NetServerMaxTickRate:90,MaxClientRate:104857600,MaxInternetClientRate:104857600,ConnectionTimeout:120,InitialConnectTimeout:120,bUseFixedFrameRate:false,bSmoothFrameRate:false,gcTimeBetweenPurging:0,useperfthreads:true,NoAsyncLoadingThread:true,UseMultithreadForDS:true,NumberOfWorkerThreadsServer:0})">高性能</n-button>
                </n-flex>
              </n-card>
              <n-card size="small" class="mb-3" title="网络">
                <div class="flex flex-col divide-y-line">
                  <div v-for="nf in [
                    {k:'NetServerMaxTickRate',l:'服务器 Tick 率',d:'每秒服务器逻辑更新次数，影响游戏流畅度与网络同步精度。0 = 使用引擎默认值，建议 60–120。',mn:0,mx:120},
                    {k:'MaxClientRate',       l:'局域网客户端带宽上限',d:'本地局域网客户端的最大数据传输速率（字节/秒）。0 = 使用引擎默认值，推荐 104857600（100 MB/s）。',mn:0,mx:209715200},
                    {k:'MaxInternetClientRate',l:'互联网客户端带宽上限',d:'互联网客户端的最大数据传输速率（字节/秒）。0 = 使用引擎默认值，推荐 104857600（100 MB/s）。',mn:0,mx:209715200},
                    {k:'ConnectionTimeout',  l:'连接超时时间（秒）',d:'已建立连接的客户端若在此时间内无响应则断开。0 = 使用引擎默认值，推荐 120。',mn:0,mx:600},
                    {k:'InitialConnectTimeout',l:'初始握手超时（秒）',d:'客户端首次建立连接时的最大等待时间。0 = 使用引擎默认值，推荐 120。',mn:0,mx:600},
                  ]" :key="nf.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ nf.l }}</span>
                        <span class="gcfg-field-key">{{ nf.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ nf.d }}</p>
                    </div>
                    <n-input-number v-model:value="ef[nf.k]" :min="nf.mn" :max="nf.mx" size="small" style="width:150px;flex-shrink:0"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="帧率 / 内存 / 多线程">
                <div class="flex flex-col divide-y-line">
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">固定帧率</span>
                        <span class="gcfg-field-key">bUseFixedFrameRate</span>
                      </div>
                      <p class="gcfg-field-desc">锁定服务器逻辑帧率为固定值，开启后须配合下方「固定帧率值」使用。不推荐与平滑帧率同时开启。</p>
                    </div>
                    <n-switch v-model:value="ef.bUseFixedFrameRate" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">固定帧率值（FPS）</span>
                        <span class="gcfg-field-key">FixedFrameRate</span>
                      </div>
                      <p class="gcfg-field-desc">启用固定帧率时生效，服务器将以此帧率运行。推荐值：30（低负载）或 60（高配置）。</p>
                    </div>
                    <n-input-number v-model:value="ef.FixedFrameRate" :min="20" :max="120" size="small" style="width:110px;flex-shrink:0" :disabled="!ef.bUseFixedFrameRate"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">平滑帧率</span>
                        <span class="gcfg-field-key">bSmoothFrameRate</span>
                      </div>
                      <p class="gcfg-field-desc">允许引擎在帧率波动时进行平滑插值，可减少卡顿感。与固定帧率互斥，专用服务器通常关闭。</p>
                    </div>
                    <n-switch v-model:value="ef.bSmoothFrameRate" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">GC 清理间隔（秒）</span>
                        <span class="gcfg-field-key">gc.TimeBetweenPurgingPendingKillObjects</span>
                      </div>
                      <p class="gcfg-field-desc">垃圾回收清理等待销毁对象的间隔时间。增大可减少 GC 频率（降低 CPU 抖动），但会提高内存占用。0 = 使用引擎默认值。</p>
                    </div>
                    <n-input-number v-model:value="ef.gcTimeBetweenPurging" :min="0" :max="600" size="small" style="width:110px;flex-shrink:0"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">性能线程优化</span>
                        <span class="gcfg-field-key">useperfthreads</span>
                      </div>
                      <p class="gcfg-field-desc">启用高性能线程调度策略，让引擎线程与 CPU 核心的绑定更高效。推荐多核服务器开启。</p>
                    </div>
                    <n-switch v-model:value="ef.useperfthreads" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">禁用异步加载线程</span>
                        <span class="gcfg-field-key">NoAsyncLoadingThread</span>
                      </div>
                      <p class="gcfg-field-desc">关闭后台异步资源加载线程，强制同步加载。可消除部分随机卡顿，但会增加主线程负担，低核心数服务器慎用。</p>
                    </div>
                    <n-switch v-model:value="ef.NoAsyncLoadingThread" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">专用服务器多线程</span>
                        <span class="gcfg-field-key">UseMultithreadForDS</span>
                      </div>
                      <p class="gcfg-field-desc">为专用服务器（Dedicated Server）启用多线程处理，充分利用多核 CPU。推荐 4 核以上服务器开启。</p>
                    </div>
                    <n-switch v-model:value="ef.UseMultithreadForDS" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">工作线程数量</span>
                        <span class="gcfg-field-key">NumberOfWorkerThreadsServer</span>
                      </div>
                      <p class="gcfg-field-desc">手动指定服务器工作线程数。0 = 由引擎根据 CPU 核心数自动决定。通常无需手动设置，除非需要限制线程占用。</p>
                    </div>
                    <n-input-number v-model:value="ef.NumberOfWorkerThreadsServer" :min="0" :max="128" size="small" style="width:110px;flex-shrink:0"/>
                  </div>
                </div>
              </n-card>
            </n-spin>
          </div>
          <div v-else class="gcfg-pane">
            <n-input v-model:value="rawContent.engine" type="textarea" :rows="30" style="font-family:monospace;font-size:12px"/>
          </div>
        </n-scrollbar>
      </n-tab-pane>

      <!-- PalDefender -->
      <n-tab-pane name="paldefender" tab="PalDefender">
        <n-scrollbar style="max-height:calc(92vh - 155px)">
          <div class="gcfg-pane" v-if="!rawMode.paldefender">
            <n-spin :show="loading">
              <n-card size="small" class="mb-3" title="REST API">
                <div class="flex flex-col divide-y-line">
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">启用 REST API</span>
                        <span class="gcfg-field-key">EnableRestAPI</span>
                      </div>
                      <p class="gcfg-field-desc">启用后可在玩家分页点击玩家查看其帕鲁与背包。变更需重启服务器才会生效。</p>
                    </div>
                    <n-switch v-model:value="pd.EnableRestAPI" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">REST API 端口</span>
                        <span class="gcfg-field-key">RestAPIPort</span>
                      </div>
                      <p class="gcfg-field-desc">PalDefender REST API 的监听端口，默认 17993。需确保防火墙未屏蔽此端口。</p>
                    </div>
                    <n-input-number v-model:value="pd.RestAPIPort" :min="1024" :max="65535" size="small" style="width:120px;flex-shrink:0"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="MOTD（欢迎消息）">
                <p class="gcfg-field-desc" style="margin-bottom:8px">玩家进入服务器时显示的欢迎消息。支持变量：<code>{PlayerName}</code>（玩家名）、<code>{ServerName}</code>（服务器名）。每行为一条独立消息。</p>
                <n-input :value="Array.isArray(pd.MOTD)?pd.MOTD.join('\n'):(pd.MOTD||'')" @update:value="v=>pd.MOTD=v.split('\n')" type="textarea" :rows="3" size="small" placeholder="每行一条消息，支持 {PlayerName} {ServerName}"/>
              </n-card>
              <n-card size="small" class="mb-3" title="反外挂处置">
                <div class="flex flex-col divide-y-line">
                  <div v-for="sw in [
                    {k:'shouldWarnCheaters',     l:'警告作弊者',       d:'检测到作弊行为时向该玩家发送警告消息。建议开启以提醒误判玩家。'},
                    {k:'shouldWarnCheatersReason',l:'警告时附上原因',   d:'警告消息中附带触发检测的具体原因，帮助玩家了解被标记的行为。'},
                    {k:'shouldKickCheaters',     l:'自动踢出作弊者',   d:'检测到作弊时自动将玩家踢出服务器。建议先观察误判率后再开启。'},
                    {k:'shouldBanCheaters',      l:'自动封禁作弊者',   d:'检测到作弊时自动将玩家永久封禁（加入 BanList）。请谨慎开启，确认误判率较低后使用。'},
                    {k:'shouldIPBanCheaters',    l:'自动 IP 封禁作弊者',d:'检测到作弊时同时封禁其 IP 地址。注意：IP 封禁可能误伤同一 IP 下的其他玩家（如 NAT 共享网络）。',warn:true},
                  ]" :key="sw.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ sw.d }}</p>
                      <p v-if="sw.warn" class="gcfg-field-warn">⚠ IP 封禁可能误伤同一 IP 下的其他玩家。</p>
                    </div>
                    <n-switch v-model:value="pd[sw.k]" size="small"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="漏洞防护">
                <div class="flex flex-col divide-y-line">
                  <div v-for="sw in [
                    {k:'steamidProtection',           l:'防止重复 UserId 登录',     d:'阻止同一 UserId（Steam ID）在服务器同时存在多个连接，防止账号克隆攻击。'},
                    {k:'blockTowerBossCapture',        l:'禁止捕捉塔主',             d:'禁止玩家使用帕鲁球捕捉各地区塔主（Boss），防止利用 Boss 捕捉漏洞。'},
                    {k:'disableIllegalItemProtection', l:'停用非法道具防护',          d:'关闭后模组/调试道具将不再被拦截。一般不建议停用，仅在特定模组服务器需要兼容时使用。', warn:true},
                    {k:'doActionUponIllegalPalStats',  l:'自动处理异常帕鲁数值',      d:'检测到帕鲁属性数值异常（如超出上限）时自动执行处置操作，配合 palStatsMaxRank 使用。'},
                  ]" :key="sw.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ sw.d }}</p>
                      <p v-if="sw.warn" class="gcfg-field-warn">⚠ 关闭后模组/调试道具将不再被拦截，一般不建议停用。</p>
                    </div>
                    <n-switch v-model:value="pd[sw.k]" size="small"/>
                  </div>
                  <div v-for="nf in [
                    {k:'palStatsMaxRank',         l:'帕鲁强化等级上限',        d:'帕鲁允许的最大强化等级，超出此值视为异常并触发处置。-1 = 自动检测游戏上限。', mn:-1, mx:20, st:1},
                    {k:'pvpMaxToBuildingDamage',  l:'PvP 对建筑最大伤害',      d:'PvP 模式下玩家对建筑物造成的单次伤害上限。0 = 不限制。可防止高倍率武器瞬间摧毁建筑。', mn:0, mx:100000, st:1},
                    {k:'pvpMaxToPlayerDamage',    l:'PvP 对玩家最大伤害',      d:'PvP 模式下玩家对玩家造成的单次伤害上限（Beta 功能）。0 = 不限制。', mn:0, mx:100000, st:1},
                    {k:'pvpMaxToPalDamage',       l:'PvP 对帕鲁最大伤害',      d:'PvP 模式下玩家对帕鲁造成的单次伤害上限。0 = 不限制。', mn:0, mx:100000, st:1},
                    {k:'pveMaxToPalBanThreshold', l:'PvE 帕鲁伤害封禁阈值',    d:'PvE 模式下玩家对帕鲁造成的单次伤害超过此值时触发封禁处置。0 = 关闭此检测。', mn:0, mx:1000000, st:1},
                    {k:'treeLimiter',             l:'砍树速率限制（秒/棵）',    d:'限制每棵树的最短破坏时间，防止火箭/高速武器快速砍树造成服务器大量卡顿。0 = 关闭限制。', mn:0, mx:5, st:0.05},
                  ]" :key="nf.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ nf.l }}</span>
                        <span class="gcfg-field-key">{{ nf.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ nf.d }}</p>
                    </div>
                    <n-input-number v-model:value="pd[nf.k]" :min="nf.mn" :max="nf.mx" :step="nf.st" size="small" style="width:120px;flex-shrink:0"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="白名单与管理员">
                <div class="flex flex-col divide-y-line">
                  <div v-for="sw in [
                    {k:'useWhitelist',               l:'启用白名单 (WhiteList.json)',          d:'开启后仅白名单内的玩家可以加入服务器，未列入白名单的玩家将被拒绝连接。'},
                    {k:'useAdminWhitelist',           l:'启用管理员 IP 白名单',                 d:'需在 adminIPs 中设置允许的 IP 地址（通过原始文件编辑）。官方建议开启以防止管理员命令被滥用。'},
                    {k:'adminAutoLogin',              l:'白名单管理员自动登录管理模式',           d:'白名单管理员加入服务器时自动进入管理员模式，无需手动输入管理员密码。'},
                    {k:'preventAdminPasswordInChat',  l:'防止管理员密码在聊天中外泄',            d:'自动屏蔽聊天中疑似管理员密码的内容，防止误发密码到公共频道。'},
                    {k:'allowAdminCheats',            l:'允许管理员使用作弊指令',                d:'允许管理员使用 godmode、飞行等调试指令。开启后管理员拥有更高的服务器控制权限。'},
                    {k:'allowGodmodeOnehit',          l:'Godmode 模式可一击击杀',               d:'管理员在 godmode 下攻击可一击消灭任何目标。仅在 allowAdminCheats 开启时生效。'},
                  ]" :key="sw.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ sw.d }}</p>
                    </div>
                    <n-switch v-model:value="pd[sw.k]" size="small"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="聊天">
                <div class="flex flex-col divide-y-line">
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">移除聊天冷却时间</span>
                        <span class="gcfg-field-key">chatBypassWait</span>
                      </div>
                      <p class="gcfg-field-desc">移除游戏原生的聊天发送冷却限制，允许玩家连续快速发送消息。可搭配 chatMessageMaxLen 一起使用。</p>
                    </div>
                    <n-switch v-model:value="pd.chatBypassWait" size="small"/>
                  </div>
                  <div class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">聊天消息长度上限</span>
                        <span class="gcfg-field-key">chatMessageMaxLen</span>
                      </div>
                      <p class="gcfg-field-desc">单条聊天消息允许的最大字符数，超出部分将被截断。默认 128，可适当调高以支持较长消息。</p>
                    </div>
                    <n-input-number v-model:value="pd.chatMessageMaxLen" :min="1" :max="1000" size="small" style="width:110px;flex-shrink:0"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="公告">
                <div class="flex flex-col divide-y-line">
                  <div v-for="sw in [
                    {k:'announceConnections',          l:'公告玩家上下线',          d:'玩家加入或离开服务器时在聊天中广播通知全体在线玩家。'},
                    {k:'dontAnnounceAdminConnections', l:'不公告管理员上下线',      d:'管理员连接/断开时不触发上下线公告，避免暴露管理员的在线状态。'},
                    {k:'announcePunishments',          l:'公告作弊处罚',            d:'玩家被踢出或封禁时在聊天中公开通知，起到警示作用。'},
                    {k:'announcePlayerDeaths',         l:'公告玩家死亡',            d:'玩家死亡时在聊天中广播死亡消息，适合 PvP 或 Hardcore 服务器。'},
                    {k:'announceOpenOilrigBoxes',      l:'公告钻油平台宝箱开启',    d:'玩家开启钻油平台目标宝箱时全服广播，增加 PvP 争夺感。'},
                    {k:'announceHelicopterKills',      l:'公告直升机击杀',          d:'玩家击落直升机时全服广播，用于记录重要 PvP 事件。'},
                    {k:'announcePlayerSummons',        l:'公告玩家召唤帕鲁',        d:'玩家召唤特定帕鲁时全服广播，可用于监控异常召唤行为。'},
                    {k:'announceAdminSummons',         l:'公告管理员召唤帕鲁',      d:'管理员召唤帕鲁时全服广播，便于玩家了解管理员操作。'},
                  ]" :key="sw.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ sw.d }}</p>
                    </div>
                    <n-switch v-model:value="pd[sw.k]" size="small"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="日志">
                <div class="flex flex-col divide-y-line">
                  <div v-for="sw in [
                    {k:'logChat',           l:'记录聊天消息',             d:'将所有玩家的公共聊天内容记录到服务器日志文件中，便于事后审查。'},
                    {k:'logRCON',           l:'记录 RCON 指令',           d:'记录通过 RCON 接口执行的所有管理命令，用于审计管理员操作。'},
                    {k:'logPlayerLogins',   l:'记录玩家上下线',            d:'记录玩家的每次登录和登出事件，包含时间戳及玩家信息。'},
                    {k:'logPlayerDeaths',   l:'记录玩家死亡',              d:'记录玩家的死亡事件及死亡原因，可用于分析异常死亡行为。'},
                    {k:'logPlayerBuildings',l:'记录玩家建筑操作',          d:'记录玩家建造、取消建造和拆除建筑的操作记录。'},
                    {k:'logPlayerSummons',  l:'记录玩家召唤帕鲁',         d:'记录玩家召唤帕鲁的事件，可配合 announcePlayerSummons 监控异常行为。'},
                    {k:'logPlayerCaptures', l:'记录玩家捕捉帕鲁',         d:'记录玩家使用帕鲁球捕捉帕鲁的事件，含目标帕鲁信息。'},
                    {k:'logCraftings',      l:'记录玩家制作',              d:'记录玩家的物品制作记录，可用于检查异常大量制作行为。'},
                    {k:'logTechUnlocks',    l:'记录科技解锁',              d:'记录玩家解锁科技树节点的事件。'},
                    {k:'logPlayerUID',      l:'日志中记录玩家 UserId',     d:'在所有日志条目中附带玩家的 UserId（Steam ID），便于跨事件关联追踪。'},
                    {k:'logPlayerIP',       l:'日志中记录玩家 IP',         d:'在日志中记录玩家的 IP 地址。注意隐私合规：记录 IP 前请确认符合当地法规要求。'},
                    {k:'logNetworking',     l:'记录客户端网络数据',         d:'记录客户端的网络数据包信息，用于排查网络问题。会产生大量日志，通常仅在调试时开启。'},
                  ]" :key="sw.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ sw.d }}</p>
                    </div>
                    <n-switch v-model:value="pd[sw.k]" size="small"/>
                  </div>
                </div>
              </n-card>
              <n-card size="small" class="mb-3" title="其他">
                <div class="flex flex-col divide-y-line">
                  <div v-for="sw in [
                    {k:'exitServerOnStartupFailure',l:'PalDefender 启动失败时关闭服务器', d:'保护存档不在无反外挂的情况下运行。注意：若同时开启「崩溃后自动重启」，可能造成无限重启循环，建议二选一。', warn:true},
                    {k:'disableButchering',         l:'停用屠宰',                        d:'禁止玩家屠宰帕鲁，防止利用屠宰机制刷取材料或规避帕鲁死亡永久消失的惩罚。'},
                    {k:'disableRenaming',           l:'停用角色改名',                    d:'禁止玩家修改角色名称，防止玩家通过改名逃避追踪或冒充他人。'},
                    {k:'disablePalRenaming',        l:'停用帕鲁改名',                    d:'禁止玩家修改帕鲁名称，防止利用帕鲁命名进行内容违规。'},
                    {k:'RCONUsePacketIdFix',        l:'修正 RCON 封包 ID 问题',           d:'修复部分 RCON 客户端因封包 ID 顺序不符导致的通信异常。推荐开启以提升 RCON 稳定性。'},
                  ]" :key="sw.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ sw.l }}</span>
                        <span class="gcfg-field-key">{{ sw.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ sw.d }}</p>
                      <p v-if="sw.warn" class="gcfg-field-warn">⚠ 若同时开启「崩溃后自动重启」，可能造成无限重启循环，建议二选一。</p>
                    </div>
                    <n-switch v-model:value="pd[sw.k]" size="small"/>
                  </div>
                  <div v-for="nf in [
                    {k:'OilrigGoalBoxLocktime',l:'钻油平台宝箱锁定时间（秒）', d:'占领钻油平台后目标宝箱的锁定时间，期间其他玩家无法开启。0 = 无锁定。默认 300 秒。', mn:0, mx:3600, st:1},
                    {k:'RCONTimeout',          l:'RCON 连接超时（秒）',        d:'RCON 客户端响应超时时间，超时后服务器主动断开连接。推荐值 31 秒。', mn:1, mx:60, st:0.5},
                  ]" :key="nf.k" class="gcfg-detail-row">
                    <div class="gcfg-detail-info">
                      <div class="gcfg-field-meta">
                        <span class="gcfg-field-label">{{ nf.l }}</span>
                        <span class="gcfg-field-key">{{ nf.k }}</span>
                      </div>
                      <p class="gcfg-field-desc">{{ nf.d }}</p>
                    </div>
                    <n-input-number v-model:value="pd[nf.k]" :min="nf.mn" :max="nf.mx" :step="nf.st" size="small" style="width:120px;flex-shrink:0"/>
                  </div>
                </div>
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
.gcfg-cat-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}
.gcfg-cat-btn {
  padding: 6px 16px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
  border: 2px solid rgba(128,128,128,0.35);
  background: rgba(128,128,128,0.08);
  color: var(--n-text-color, inherit);
  cursor: pointer;
  transition: all 0.2s;
  &:hover {
    border-color: #18a058;
    color: #18a058;
    background: rgba(24,160,88,0.08);
  }
  &.active {
    background: #18a058;
    color: #fff !important;
    border-color: #18a058;
  }
}
.gcfg-section-title {
  font-size: 12px;
  font-weight: 700;
  color: rgba(128,128,128,0.7);
  margin: 14px 0 8px;
  padding-bottom: 4px;
  border-bottom: 1px solid rgba(128,128,128,0.1);
}
.gcfg-label { font-size: 13px; }
.gcfg-toggle-row {
  padding: 5px 0;
  border-bottom: 1px solid rgba(128,128,128,0.1);
}
.gcfg-slider-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px 0;
  border-bottom: 1px solid rgba(128,128,128,0.08);
}
.gcfg-slider-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.gcfg-slider-label {
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.gcfg-field-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
  margin-bottom: 2px;
}
.gcfg-field-label {
  font-size: 13px;
  font-weight: 700;
}
.gcfg-field-key {
  font-family: 'Menlo', 'Consolas', monospace;
  font-size: 11px;
  color: rgba(128,128,128,0.75);
}
.gcfg-field-desc {
  font-size: 12px;
  color: rgba(128,128,128,0.8);
  margin: 2px 0 0;
  line-height: 1.5;
}
.gcfg-field-warn {
  font-size: 11px;
  color: #d48806;
  margin: 4px 0 0;
  line-height: 1.5;
}
.gcfg-toggle-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gcfg-detail-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid rgba(128,128,128,0.1);
  &:last-child { border-bottom: none; }
}
.gcfg-detail-info {
  flex: 1;
  min-width: 0;
}
.divide-y-line > .gcfg-detail-row:last-child {
  border-bottom: none;
}
</style>
