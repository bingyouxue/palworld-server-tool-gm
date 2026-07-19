<script setup>
import { ref, reactive, nextTick, onMounted } from "vue";
import { useMessage, useDialog } from "naive-ui";
const message = useMessage();
const dialog = useDialog();
const TOKEN_KEY = "palworld_token";

const show      = ref(false);
const step      = ref(1);   // 1=choose  2=adopt|install  3=world-settings
const mode      = ref("");  // "adopt"|"install"

// Actual server root reported by the backend after install (may differ from installDir)
const installedServerDir = ref("");

// Step 2A – adopt
const adoptDir     = ref("");
const adoptParsed  = ref(null);
const adoptLoading = ref(false);

// Step 2B – install
const installDir  = ref("");
const installLog  = ref([]);
const installing  = ref(false);
const installDone = ref(false);

// Step 3 – world settings
const worldSaving = ref(false);
const WS_DEFAULTS = {
  ServerName: 'Default Palworld Server', ServerDescription: '', AdminPassword: '', ServerPassword: '',
  ServerPlayerMaxNum: '32', CoopPlayerMaxNum: '4', GuildPlayerMaxNum: '20',
  PublicPort: '8211', PublicIP: '', Region: '', bUseAuth: 'True',
  BanListURL: 'https://b.palworldgame.com/api/banlist.txt',
  RCONEnabled: 'True', RCONPort: '25575', RESTAPIEnabled: 'True', RESTAPIPort: '8212',
  bIsMultiplay: 'False', bIsPvP: 'False', bShowPlayerList: 'False',
  bIsShowJoinLeftMessage: 'True', bIsUseBackupSaveData: 'True',
  bAllowClientMod: 'True', CrossplayPlatforms: '(Steam,Xbox,PS5,Mac)',
  LogFormatType: 'Text', ChatPostLimitPerMinute: '30',
  bEnableVoiceChat: 'False', VoiceChatMaxVolumeDistance: '3000.000000', VoiceChatZeroVolumeDistance: '15000.000000',
  Difficulty: 'None', RandomizerType: 'None', RandomizerSeed: '', bIsRandomizerPalLevelRandom: 'False',
  AutoSaveSpan: '30.000000',
  bHardcore: 'False', bPalLost: 'False', bCharacterRecreateInHardcore: 'False',
  DayTimeSpeedRate: '1.000000', NightTimeSpeedRate: '1.000000',
  ExpRate: '1.000000', WorkSpeedRate: '1.000000',
  PalCaptureRate: '1.000000', PalSpawnNumRate: '1.000000',
  PalDamageRateAttack: '1.000000', PalDamageRateDefense: '1.000000',
  PalStomachDecreaceRate: '1.000000', PalStaminaDecreaceRate: '1.000000',
  PalAutoHPRegeneRate: '1.000000', PalAutoHpRegeneRateInSleep: '1.000000',
  PalEggDefaultHatchingTime: '1.000000',
  PlayerDamageRateAttack: '1.000000', PlayerDamageRateDefense: '1.000000',
  PlayerStomachDecreaceRate: '1.000000', PlayerStaminaDecreaceRate: '1.000000',
  PlayerAutoHPRegeneRate: '1.000000', PlayerAutoHpRegeneRateInSleep: '1.000000',
  ItemWeightRate: '1.000000', EquipmentDurabilityDamageRate: '1.000000',
  BuildObjectHpRate: '1.000000', BuildObjectDamageRate: '1.000000',
  BuildObjectDeteriorationDamageRate: '1.000000',
  CollectionDropRate: '1.000000', CollectionObjectHpRate: '1.000000',
  CollectionObjectRespawnSpeedRate: '1.000000', EnemyDropItemRate: '1.000000',
  DropItemMaxNum: '3000', PhysicsActiveDropItemMaxNum: '-1',
  DropItemMaxNum_UNKO: '100', DropItemAliveMaxHours: '1.000000',
  MaxBuildingLimitNum: '0', ServerReplicatePawnCullDistance: '15000.000000',
  MonsterFarmActionSpeedRate: '1.000000', SupplyDropSpan: '180',
  DeathPenalty: 'Item',
  bCanPickupOtherGuildDeathPenaltyDrop: 'False', bEnableNonLoginPenalty: 'True',
  bEnablePlayerToPlayerDamage: 'False', bEnableFriendlyFire: 'False',
  bDisplayPvPItemNumOnWorldMap_BaseCamp: 'False', bDisplayPvPItemNumOnWorldMap_Player: 'False',
  AdditionalDropItemWhenPlayerKillingInPvPMode: 'PlayerDropItem',
  AdditionalDropItemNumWhenPlayerKillingInPvPMode: '1',
  bAdditionalDropItemWhenPlayerKillingInPvPMode: 'False',
  BaseCampMaxNum: '128', BaseCampMaxNumInGuild: '4', BaseCampWorkerMaxNum: '15',
  GuildRejoinCooldownMinutes: '0',
  bAutoResetGuildNoOnlinePlayers: 'False', AutoResetGuildTimeNoOnlinePlayers: '72.000000',
  AutoTransferMasterCheckIntervalSeconds: '3600.000000', AutoTransferMasterThresholdDays: '14',
  MaxGuildsPerFrame: '10',
  bEnableInvaderEnemy: 'True', EnablePredatorBossPal: 'True',
  bActiveUNKO: 'False', bEnableAimAssistPad: 'True', bEnableAimAssistKeyboard: 'False',
  bEnableFastTravel: 'True', bEnableFastTravelOnlyBaseCamp: 'False',
  bIsStartLocationSelectByMap: 'False', bExistPlayerAfterLogout: 'False',
  bEnableDefenseOtherGuildPlayer: 'False', bInvisibleOtherGuildBaseCampAreaFX: 'False',
  bBuildAreaLimit: 'False',
  bAllowGlobalPalboxExport: 'True', bAllowGlobalPalboxImport: 'False',
  bAllowEnhanceStat_Health: 'True', bAllowEnhanceStat_Attack: 'True',
  bAllowEnhanceStat_Stamina: 'True', bAllowEnhanceStat_Weight: 'True',
  bAllowEnhanceStat_WorkSpeed: 'True',
  BlockRespawnTime: '5.000000', RespawnPenaltyDurationThreshold: '0.000000', RespawnPenaltyTimeScale: '2.000000',
  DenyTechnologyList: '',
  ItemContainerForceMarkDirtyInterval: '1.000000',
  PlayerDataPalStorageUpdateCheckTickInterval: '1.000000',
  ItemCorruptionMultiplier: '1.000000',
  bEnableBuildingPlayerUIdDisplay: 'False', BuildingNameDisplayCacheTTLSeconds: '60',
};
const ws = reactive({ ...WS_DEFAULTS });
function resetWorldSettings() { Object.assign(ws, WS_DEFAULTS); }
const WS_ALIASES = {
  PlayerStomachDecreaseRate: 'PlayerStomachDecreaceRate',
  PlayerStaminaDecreaseRate: 'PlayerStaminaDecreaceRate',
  PalStomachDecreaseRate:    'PalStomachDecreaceRate',
  PalStaminaDecreaseRate:    'PalStaminaDecreaceRate',
};
function fillWorldSettings(parsed) {
  resetWorldSettings();
  if (parsed?.world_settings) {
    Object.entries(parsed.world_settings).forEach(([k, v]) => {
      const key = WS_ALIASES[k] || k;
      if (key in ws) ws[key] = v;
    });
  }
  // PST relies on these management interfaces. New/adopted servers default
  // them to enabled even when an old INI explicitly disabled them.
  ws.RCONEnabled = 'True';
  ws.RESTAPIEnabled = 'True';
  if (parsed?.admin_password) ws.AdminPassword = parsed.admin_password;
  if (parsed?.rcon_port)      ws.RCONPort = String(parsed.rcon_port);
  if (parsed?.rest_port)      ws.RESTAPIPort = String(parsed.rest_port);
}

const WS_GROUPS = [
  { label: "服务器基本", keys: ["ServerName","ServerDescription","AdminPassword","ServerPassword","ServerPlayerMaxNum","CoopPlayerMaxNum","GuildPlayerMaxNum","PublicPort","Difficulty","AutoSaveSpan"] },
  { label: "网络 / RCON / REST", keys: ["bIsMultiplay","bIsPvP","RCONEnabled","RCONPort","RESTAPIEnabled","RESTAPIPort","bShowPlayerList","bIsShowJoinLeftMessage","ChatPostLimitPerMinute","bAllowClientMod"] },
  { label: "语音聊天", keys: ["bEnableVoiceChat","VoiceChatMaxVolumeDistance","VoiceChatZeroVolumeDistance"] },
  { label: "随机器", keys: ["RandomizerType","RandomizerSeed","bIsRandomizerPalLevelRandom"] },
  { label: "时间 / 经验 / 工作", keys: ["DayTimeSpeedRate","NightTimeSpeedRate","ExpRate","WorkSpeedRate"] },
  { label: "帕鲁倍率", keys: ["PalCaptureRate","PalSpawnNumRate","PalDamageRateAttack","PalDamageRateDefense","PalStomachDecreaceRate","PalStaminaDecreaceRate","PalAutoHPRegeneRate","PalAutoHpRegeneRateInSleep","PalEggDefaultHatchingTime"] },
  { label: "玩家倍率", keys: ["PlayerDamageRateAttack","PlayerDamageRateDefense","PlayerStomachDecreaceRate","PlayerStaminaDecreaceRate","PlayerAutoHPRegeneRate","PlayerAutoHpRegeneRateInSleep","ItemWeightRate","EquipmentDurabilityDamageRate"] },
  { label: "建筑 / 采集 / 掉落", keys: ["BuildObjectHpRate","BuildObjectDamageRate","BuildObjectDeteriorationDamageRate","CollectionDropRate","CollectionObjectHpRate","CollectionObjectRespawnSpeedRate","EnemyDropItemRate","DropItemMaxNum","DropItemAliveMaxHours","PhysicsActiveDropItemMaxNum","DropItemMaxNum_UNKO","MaxBuildingLimitNum","MonsterFarmActionSpeedRate","SupplyDropSpan"] },
  { label: "死亡 / 惩罚", keys: ["DeathPenalty","bCanPickupOtherGuildDeathPenaltyDrop","bEnableNonLoginPenalty","BlockRespawnTime","RespawnPenaltyDurationThreshold","RespawnPenaltyTimeScale"] },
  { label: "PvP", keys: ["bEnablePlayerToPlayerDamage","bEnableFriendlyFire","bDisplayPvPItemNumOnWorldMap_BaseCamp","bDisplayPvPItemNumOnWorldMap_Player","bAdditionalDropItemWhenPlayerKillingInPvPMode","AdditionalDropItemWhenPlayerKillingInPvPMode","AdditionalDropItemNumWhenPlayerKillingInPvPMode"] },
  { label: "公会 / 营地", keys: ["BaseCampMaxNum","BaseCampMaxNumInGuild","BaseCampWorkerMaxNum","GuildRejoinCooldownMinutes","bAutoResetGuildNoOnlinePlayers","AutoResetGuildTimeNoOnlinePlayers","AutoTransferMasterCheckIntervalSeconds","AutoTransferMasterThresholdDays","MaxGuildsPerFrame"] },
  { label: "世界 / 探索", keys: ["bEnableInvaderEnemy","EnablePredatorBossPal","bActiveUNKO","bEnableAimAssistPad","bEnableAimAssistKeyboard","bEnableFastTravel","bEnableFastTravelOnlyBaseCamp","bIsStartLocationSelectByMap","bExistPlayerAfterLogout","bEnableDefenseOtherGuildPlayer","bInvisibleOtherGuildBaseCampAreaFX","bBuildAreaLimit","ServerReplicatePawnCullDistance"] },
  { label: "硬核 / 帕鲁丢失", keys: ["bHardcore","bPalLost","bCharacterRecreateInHardcore"] },
  { label: "帕鲁箱 / 属性强化", keys: ["bAllowGlobalPalboxExport","bAllowGlobalPalboxImport","bAllowEnhanceStat_Health","bAllowEnhanceStat_Attack","bAllowEnhanceStat_Stamina","bAllowEnhanceStat_Weight","bAllowEnhanceStat_WorkSpeed"] },
  { label: "高级 / 杂项", keys: ["DenyTechnologyList","ItemContainerForceMarkDirtyInterval","PlayerDataPalStorageUpdateCheckTickInterval","ItemCorruptionMultiplier","bEnableBuildingPlayerUIdDisplay","BuildingNameDisplayCacheTTLSeconds","bIsUseBackupSaveData"] },
];
const WS_LABELS = {
  ServerName:'服务器名称', ServerDescription:'服务器描述', AdminPassword:'管理员密码',
  ServerPassword:'进服密码(空=公开)', ServerPlayerMaxNum:'最大玩家数', CoopPlayerMaxNum:'合作玩家上限',
  GuildPlayerMaxNum:'公会最大人数', PublicPort:'公开端口', Difficulty:'难度', AutoSaveSpan:'自动保存间隔(s)',
  bIsMultiplay:'多人游戏', bIsPvP:'启用PvP', bShowPlayerList:'显示玩家列表',
  bIsShowJoinLeftMessage:'显示进出消息', ChatPostLimitPerMinute:'聊天每分钟上限', bAllowClientMod:'允许客户端Mod',
  RCONEnabled:'启用RCON', RCONPort:'RCON端口', RESTAPIEnabled:'启用REST API', RESTAPIPort:'REST API端口',
  bEnableVoiceChat:'启用语音聊天', VoiceChatMaxVolumeDistance:'语音最大音量距离', VoiceChatZeroVolumeDistance:'语音零音量距离',
  RandomizerType:'随机器类型', RandomizerSeed:'随机器种子', bIsRandomizerPalLevelRandom:'随机帕鲁等级',
  DayTimeSpeedRate:'白天速率', NightTimeSpeedRate:'夜晚速率', ExpRate:'经验倍率', WorkSpeedRate:'工作速率',
  PalCaptureRate:'帕鲁捕捉倍率', PalSpawnNumRate:'帕鲁生成倍率',
  PalDamageRateAttack:'帕鲁攻击倍率', PalDamageRateDefense:'帕鲁防御倍率',
  PalStomachDecreaceRate:'帕鲁饥饿率', PalStaminaDecreaceRate:'帕鲁体力消耗率',
  PalAutoHPRegeneRate:'帕鲁自动回血率', PalAutoHpRegeneRateInSleep:'帕鲁睡眠回血率',
  PalEggDefaultHatchingTime:'帕鲁蛋孵化时间(h)',
  PlayerDamageRateAttack:'玩家攻击倍率', PlayerDamageRateDefense:'玩家防御倍率',
  PlayerStomachDecreaceRate:'玩家饥饿率', PlayerStaminaDecreaceRate:'玩家体力消耗率',
  PlayerAutoHPRegeneRate:'玩家自动回血率', PlayerAutoHpRegeneRateInSleep:'玩家睡眠回血率',
  ItemWeightRate:'物品重量倍率', EquipmentDurabilityDamageRate:'装备耐久损耗倍率',
  BuildObjectHpRate:'建筑HP倍率', BuildObjectDamageRate:'建筑受伤倍率',
  BuildObjectDeteriorationDamageRate:'建筑劣化速率',
  CollectionDropRate:'采集掉落倍率', CollectionObjectHpRate:'采集物HP倍率',
  CollectionObjectRespawnSpeedRate:'采集物刷新速率', EnemyDropItemRate:'敌人掉落倍率',
  DropItemMaxNum:'掉落物最大数量', DropItemAliveMaxHours:'掉落物保留时间(h)',
  PhysicsActiveDropItemMaxNum:'物理掉落物上限(-1=无限)', DropItemMaxNum_UNKO:'UNKO掉落上限',
  MaxBuildingLimitNum:'建筑上限(0=无限)', MonsterFarmActionSpeedRate:'怪物农场速率', SupplyDropSpan:'补给投放间隔(min)',
  DeathPenalty:'死亡惩罚', bCanPickupOtherGuildDeathPenaltyDrop:'可捡取他人死亡掉落',
  bEnableNonLoginPenalty:'启用未登录惩罚',
  BlockRespawnTime:'阻止重生时间(s)', RespawnPenaltyDurationThreshold:'重生惩罚时长阈值', RespawnPenaltyTimeScale:'重生惩罚时间倍率',
  bEnablePlayerToPlayerDamage:'玩家间可造成伤害', bEnableFriendlyFire:'友伤',
  bDisplayPvPItemNumOnWorldMap_BaseCamp:'地图显示营地PvP数量', bDisplayPvPItemNumOnWorldMap_Player:'地图显示玩家PvP数量',
  bAdditionalDropItemWhenPlayerKillingInPvPMode:'PvP击杀额外掉落',
  AdditionalDropItemWhenPlayerKillingInPvPMode:'PvP击杀掉落类型', AdditionalDropItemNumWhenPlayerKillingInPvPMode:'PvP击杀掉落数量',
  BaseCampMaxNum:'营地最大数量', BaseCampMaxNumInGuild:'公会营地上限', BaseCampWorkerMaxNum:'营地工作帕鲁上限',
  GuildRejoinCooldownMinutes:'公会重新加入冷却(min)',
  bAutoResetGuildNoOnlinePlayers:'无人在线自动重置公会', AutoResetGuildTimeNoOnlinePlayers:'无人在线重置时间(h)',
  AutoTransferMasterCheckIntervalSeconds:'自动移交检测间隔(s)', AutoTransferMasterThresholdDays:'自动移交阈值(天)', MaxGuildsPerFrame:'每帧最大公会数',
  bEnableInvaderEnemy:'启用突袭', EnablePredatorBossPal:'启用顶点帕鲁',
  bActiveUNKO:'启用UNKO', bEnableAimAssistPad:'手柄辅助瞄准', bEnableAimAssistKeyboard:'键盘辅助瞄准',
  bEnableFastTravel:'启用快速旅行', bEnableFastTravelOnlyBaseCamp:'仅营地快速旅行',
  bIsStartLocationSelectByMap:'可选出生点', bExistPlayerAfterLogout:'退出后角色保留',
  bEnableDefenseOtherGuildPlayer:'可攻击他会玩家', bInvisibleOtherGuildBaseCampAreaFX:'隐藏他会领地特效',
  bBuildAreaLimit:'限制建筑区域', ServerReplicatePawnCullDistance:'服务器剔除距离',
  bHardcore:'硬核模式', bPalLost:'帕鲁死亡丢失', bCharacterRecreateInHardcore:'硬核模式角色重建',
  bAllowGlobalPalboxExport:'允许全局帕鲁箱导出', bAllowGlobalPalboxImport:'允许全局帕鲁箱导入',
  bAllowEnhanceStat_Health:'允许强化生命', bAllowEnhanceStat_Attack:'允许强化攻击',
  bAllowEnhanceStat_Stamina:'允许强化体力', bAllowEnhanceStat_Weight:'允许强化负重',
  bAllowEnhanceStat_WorkSpeed:'允许强化工作速度',
  DenyTechnologyList:'禁用科技列表', ItemContainerForceMarkDirtyInterval:'容器强制标脏间隔(s)',
  PlayerDataPalStorageUpdateCheckTickInterval:'帕鲁存储更新间隔(s)', ItemCorruptionMultiplier:'物品腐蚀倍率',
  bEnableBuildingPlayerUIdDisplay:'显示建筑所有者', BuildingNameDisplayCacheTTLSeconds:'建筑名称缓存时间(s)',
  bIsUseBackupSaveData:'启用备份存档',
};

function isBool(k) { const v = ws[k]; return v === 'True' || v === 'False'; }
function boolVal(k)   { return ws[k] === 'True'; }
function setBool(k,v) { ws[k] = v ? 'True' : 'False'; }
function isSelect(k)  {
  return ['DeathPenalty','Difficulty','RandomizerType','AdditionalDropItemWhenPlayerKillingInPvPMode','LogFormatType'].includes(k);
}
const NUMERIC_KEYS = new Set([
  'DayTimeSpeedRate','NightTimeSpeedRate','ExpRate','WorkSpeedRate',
  'PalCaptureRate','PalSpawnNumRate','PalDamageRateAttack','PalDamageRateDefense',
  'PalStomachDecreaceRate','PalStaminaDecreaceRate','PalAutoHPRegeneRate','PalAutoHpRegeneRateInSleep',
  'PalEggDefaultHatchingTime',
  'PlayerDamageRateAttack','PlayerDamageRateDefense','PlayerStomachDecreaceRate','PlayerStaminaDecreaceRate',
  'PlayerAutoHPRegeneRate','PlayerAutoHpRegeneRateInSleep',
  'ItemWeightRate','EquipmentDurabilityDamageRate',
  'BuildObjectHpRate','BuildObjectDamageRate','BuildObjectDeteriorationDamageRate',
  'CollectionDropRate','CollectionObjectHpRate','CollectionObjectRespawnSpeedRate','EnemyDropItemRate',
  'DropItemAliveMaxHours','MonsterFarmActionSpeedRate',
  'AutoResetGuildTimeNoOnlinePlayers','AutoTransferMasterCheckIntervalSeconds',
  'AutoSaveSpan','BlockRespawnTime','RespawnPenaltyDurationThreshold','RespawnPenaltyTimeScale',
  'VoiceChatMaxVolumeDistance','VoiceChatZeroVolumeDistance','ServerReplicatePawnCullDistance',
  'ItemContainerForceMarkDirtyInterval','PlayerDataPalStorageUpdateCheckTickInterval','ItemCorruptionMultiplier',
]);
const INT_KEYS = new Set([
  'ServerPlayerMaxNum','CoopPlayerMaxNum','GuildPlayerMaxNum','PublicPort','RCONPort','RESTAPIPort',
  'DropItemMaxNum','PhysicsActiveDropItemMaxNum','DropItemMaxNum_UNKO','MaxBuildingLimitNum',
  'BaseCampMaxNum','BaseCampMaxNumInGuild','BaseCampWorkerMaxNum','SupplyDropSpan',
  'AutoTransferMasterThresholdDays','MaxGuildsPerFrame','GuildRejoinCooldownMinutes',
  'ChatPostLimitPerMinute','AdditionalDropItemNumWhenPlayerKillingInPvPMode',
  'BuildingNameDisplayCacheTTLSeconds',
]);
function isNumeric(k) { return NUMERIC_KEYS.has(k) || INT_KEYS.has(k); }
function numStep(k)   { return INT_KEYS.has(k) ? 1 : 0.1; }
function numMin(k)    { return k === 'PhysicsActiveDropItemMaxNum' ? -1 : 0; }
const selectOptions = {
  DeathPenalty: [
    {label:'None — 不掉落',value:'None'},
    {label:'Item — 掉落道具',value:'Item'},
    {label:'ItemAndEquipment — 掉落道具和装备',value:'ItemAndEquipment'},
    {label:'All — 全部掉落',value:'All'},
  ],
  Difficulty: [
    {label:'None（默认）',value:'None'},
    {label:'Casual（休闲）',value:'Casual'},
    {label:'Normal（普通）',value:'Normal'},
    {label:'Hard（困难）',value:'Hard'},
  ],
  RandomizerType: [
    {label:'None（关闭）',value:'None'},
    {label:'Default（默认随机）',value:'Default'},
  ],
  AdditionalDropItemWhenPlayerKillingInPvPMode: [
    {label:'PlayerDropItem — 玩家死亡掉落',value:'PlayerDropItem'},
    {label:'None — 不额外掉落',value:'None'},
  ],
  LogFormatType: [
    {label:'Text（文本）',value:'Text'},
    {label:'Json（JSON）',value:'Json'},
  ],
};

// ── Directory browser ──────────────────────────────────────────────────────
const showDirBrowser = ref(false);
const dirTarget  = ref("");   // "adopt"|"install"
const dirCurrent = ref("");
const dirParent  = ref("");
const dirRoots   = ref([]);
const dirEntries = ref([]);
const dirLoading = ref(false);
const dirError   = ref("");

const mkdirShow    = ref(false);
const mkdirName    = ref("");
const mkdirError   = ref("");
const mkdirInputEl = ref(null);

const authHeader = () => ({
  "Content-Type": "application/json",
  Authorization: `Bearer ${localStorage.getItem(TOKEN_KEY)||""}`,
});

async function openBrowser(target) {
  dirTarget.value  = target;
  const init = target==="adopt" ? adoptDir.value.trim() : installDir.value.trim();
  dirCurrent.value = init;
  dirParent.value  = "";
  dirRoots.value   = [];
  dirEntries.value = [];
  dirError.value   = "";
  mkdirShow.value  = false;
  showDirBrowser.value = true;
  await browseDir(init);
}

async function browseDir(path) {
  dirLoading.value = true;
  dirError.value   = "";
  try {
    const q = path ? `?path=${encodeURIComponent(path)}` : "";
    const r = await fetch(`/api/config/directories${q}`, { headers: authHeader() });
    const j = await r.json();
    if (!r.ok) {
      if (path) { await browseDir(""); return; }
      dirError.value = j.error || "无法读取目录"; return;
    }
    dirCurrent.value = j.current || path || "";
    dirParent.value  = j.parent  || "";
    dirRoots.value   = j.roots   || [];
    dirEntries.value = j.entries || [];
  } catch(e) {
    if (path) { await browseDir(""); return; }
    dirError.value = e.message;
  } finally { dirLoading.value = false; }
}

function browserUp() {
  const p = dirParent.value, c = dirCurrent.value;
  if (!p || p===c) { dirCurrent.value=""; dirParent.value=""; dirEntries.value=[]; return; }
  browseDir(p);
}
function browserEnter(e) { browseDir(e.Path||e.path||e.name); }
function browserSelectRoot(root) { browseDir(root); }
function browserConfirm() {
  if (!dirCurrent.value) return;
  if (dirTarget.value==="adopt") adoptDir.value = dirCurrent.value;
  else installDir.value = dirCurrent.value;
  showDirBrowser.value = false;
}

function startMkdir() {
  mkdirShow.value  = true;
  mkdirName.value  = "";
  mkdirError.value = "";
  nextTick(() => mkdirInputEl.value?.focus());
}
async function confirmMkdir() {
  const name = mkdirName.value.trim();
  if (!name) { mkdirError.value="请输入文件夹名称"; return; }
  const sep = dirCurrent.value.match(/^[A-Za-z]:/) ? "\\" : "/";
  const newPath = dirCurrent.value.replace(/[\/\\]+$/, "") + sep + name;
  try {
    const r = await fetch("/api/config/mkdir", { method:"POST", headers:authHeader(), body:JSON.stringify({path:newPath}) });
    const j = await r.json();
    if (!r.ok) { mkdirError.value=j.error||"创建失败"; return; }
    mkdirShow.value = false;
    await browseDir(newPath);
  } catch(e) { mkdirError.value=e.message; }
}

// ── Wizard flow ─────────────────────────────────────────────────────────────
onMounted(async () => {
  try {
    const r = await fetch("/api/setup/status");
    const j = await r.json();
    if (!j.done) show.value = true;
  } catch { show.value = true; }
});

function chooseMode(m) { mode.value=m; step.value=2; }

async function doAdopt() {
  if (!adoptDir.value.trim()) { message.warning("请输入服务器路径"); return; }
  adoptLoading.value = true;
  try {
    const r = await fetch("/api/setup/adopt", { method:"POST", headers:authHeader(), body:JSON.stringify({server_dir:adoptDir.value.trim()}) });
    const j = await r.json();
    if (!r.ok) { message.error(j.error||"验证失败"); return; }
    adoptParsed.value = j.parsed;
    fillWorldSettings(j.parsed);
    step.value = 3;
  } finally { adoptLoading.value=false; }
}

async function doInstall() {
  if (!installDir.value.trim()) { message.warning("请选择安装路径"); return; }
  installing.value=true; installLog.value=[]; installDone.value=false;
  try {
    const r = await fetch("/api/setup/install", { method:"POST", headers:authHeader(), body:JSON.stringify({install_dir:installDir.value.trim()}) });
    const j = await r.json();
    if (!r.ok) { message.error(j.error||"安装失败"); return; }
    const es = new EventSource(`/api/setup/install/progress/${j.install_id}`);
    es.addEventListener("log", e => {
      const line = e.data || "";
      // Backend emits "[server_dir] <path>" after successful install
      if (line.startsWith("[server_dir] ")) {
        installedServerDir.value = line.slice("[server_dir] ".length).trim();
      } else {
        installLog.value.push(line);
      }
    });
    es.addEventListener("done",  () => { es.close(); installing.value=false; installDone.value=true; });
    es.addEventListener("error", e => { installLog.value.push("[错误] "+(e.data||"连接断开")); es.close(); installing.value=false; installDone.value=true; });
  } catch(e) { message.error("请求失败: "+e.message); installing.value=false; }
}

async function saveWorldSettings() {
  const managementEnabled = boolVal('RCONEnabled') || boolVal('RESTAPIEnabled');
  if (managementEnabled && !String(ws.AdminPassword || '').trim()) {
    const confirmed = await new Promise((resolve) => {
      let settled = false;
      const finish = (value) => {
        if (settled) return;
        settled = true;
        resolve(value);
      };
      dialog.warning({
        title: "管理员密码未设置",
        content: "RCON 或 REST API 已启用，但 AdminPassword 为空。控制面板将无法安全认证，远程管理接口也存在安全风险。确定仍要保存吗？",
        positiveText: "仍然保存",
        negativeText: "返回填写密码",
        onPositiveClick: () => finish(true),
        onNegativeClick: () => finish(false),
        onClose: () => finish(false),
      });
    });
    if (!confirmed) return;
  }

  worldSaving.value = true;
  try {
    // For install mode, use the server root reported by the backend (installedServerDir).
    // This is the directory where PalServer actually lives, which may differ from
    // the user-chosen installDir (e.g. steamcmd places the game in a subdirectory).
    const serverDir = mode.value==="install"
      ? (installedServerDir.value || installDir.value.trim())
      : (adoptParsed.value?.server_dir||"");
    const r = await fetch("/api/setup/world-settings", { method:"PUT", headers:authHeader(), body:JSON.stringify({server_dir:serverDir, settings:{...ws}}) });
    const j = await r.json();
    if (!r.ok) { message.error(j.error||"保存失败"); return; }
    await fetch("/api/setup/complete", { method:"POST", headers:authHeader() });
    show.value = false;
    message.success("设置已保存，正在刷新...");
    setTimeout(()=>location.reload(), 1200);
  } finally { worldSaving.value=false; }
}
</script>

<template>
  <!-- ── Main wizard modal ── -->
  <n-modal :show="show" :mask-closable="false" :close-on-esc="false"
    transform-origin="center" style="width:min(640px,96vw)">
    <n-card :bordered="false" role="dialog" style="border-radius:14px">
      <template #header>
        <n-flex align="center" :size="10">
          <svg stroke="#2563eb" fill="none" stroke-width="2" viewBox="0 0 24 24"
            stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
            <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/>
          </svg>
          <span style="font-weight:800;color:#1e3a8a;font-size:15px">幻兽帕鲁服务端管理 — 初始设置</span>
          <n-flex :size="4" style="margin-left:auto;align-items:center">
            <span :class="['sdot', step>=1?'sdot-on':'']">1</span>
            <span class="sline"/>
            <span :class="['sdot', step>=2?'sdot-on':'']">2</span>
            <span class="sline"/>
            <span :class="['sdot', step>=3?'sdot-on':'']">3</span>
          </n-flex>
        </n-flex>
      </template>

      <!-- STEP 1 -->
      <div v-if="step===1">
        <p style="color:#4b5563;font-size:13px;margin:0 0 16px">欢迎！在开始前，请告诉我们你的服务器情况：</p>
        <n-grid :cols="2" :x-gap="12" :y-gap="12">
          <n-gi>
            <div class="choice-card" @click="chooseMode('adopt')">
              <svg stroke="#2563eb" fill="none" stroke-width="2" viewBox="0 0 24 24"
                stroke-linecap="round" stroke-linejoin="round" width="28" height="28">
                <rect x="2" y="3" width="20" height="14" rx="2"/>
                <line x1="8" y1="21" x2="16" y2="21"/>
                <line x1="12" y1="17" x2="12" y2="21"/>
              </svg>
              <div class="choice-label">我有现有服务器</div>
              <div class="choice-desc">指定服务器目录，自动读取现有配置</div>
            </div>
          </n-gi>
          <n-gi>
            <div class="choice-card" @click="chooseMode('install')">
              <svg stroke="#2563eb" fill="none" stroke-width="2" viewBox="0 0 24 24"
                stroke-linecap="round" stroke-linejoin="round" width="28" height="28">
                <polyline points="16 16 12 12 8 16"/>
                <line x1="12" y1="12" x2="12" y2="21"/>
                <path d="M20.39 18.39A5 5 0 0 0 18 9h-1.26A8 8 0 1 0 3 16.3"/>
              </svg>
              <div class="choice-label">帮我安装服务器</div>
              <div class="choice-desc">通过 SteamCMD 自动下载安装幻兽帕鲁服务端</div>
            </div>
          </n-gi>
        </n-grid>
      </div>

      <!-- STEP 2A: adopt -->
      <div v-else-if="step===2 && mode==='adopt'">
        <n-flex align="center" :size="8" style="margin-bottom:14px">
          <n-button text type="primary" @click="step=1;adoptParsed=null">← 返回</n-button>
          <span style="font-weight:700;color:#1e3a8a">关联现有服务器</span>
        </n-flex>
        <n-form-item label="服务器根目录（包含 PalServer 可执行文件的目录）" label-placement="top">
          <n-input-group>
            <n-input v-model:value="adoptDir" placeholder="请选择或输入路径" />
            <n-button @click="openBrowser('adopt')">浏览</n-button>
          </n-input-group>
        </n-form-item>
        <n-button type="primary" :loading="adoptLoading" @click="doAdopt">
          验证并读取配置 →
        </n-button>
      </div>

      <!-- STEP 2B: install -->
      <div v-else-if="step===2 && mode==='install'">
        <n-flex align="center" :size="8" style="margin-bottom:14px">
          <n-button text type="primary" @click="step=1">← 返回</n-button>
          <span style="font-weight:700;color:#1e3a8a">通过 SteamCMD 安装服务器</span>
        </n-flex>
        <n-form-item label="安装目录" label-placement="top">
          <n-input-group>
            <n-input v-model:value="installDir" :disabled="installing" placeholder="请选择安装目录" />
            <n-button :disabled="installing" @click="openBrowser('install')">浏览</n-button>
          </n-input-group>
          <template #feedback>选择一个空目录，SteamCMD 将把服务端文件（约 12 GB）安装到此。注：SteamCMD安装时间较长请耐心等待</template>
        </n-form-item>
        <n-button type="primary" :loading="installing" :disabled="installing||installDone" @click="doInstall">
          {{ installDone ? '安装完成' : '开始安装' }}
        </n-button>
        <div v-if="installLog.length" class="install-log">
          <div v-for="(line,i) in installLog" :key="i"
            :style="line.startsWith('[错误]')?{color:'#dc2626'}:{}">{{ line }}</div>
        </div>
        <n-button v-if="installDone && !installLog.some(l=>l.startsWith('[错误]'))"
          type="primary" style="margin-top:12px" @click="step=3">
          设置世界参数 →
        </n-button>
      </div>

      <!-- STEP 3: world settings -->
      <div v-else-if="step===3">
        <n-flex align="center" :size="8" style="margin-bottom:14px;flex-wrap:wrap">
          <n-button text type="primary" @click="step=2">← 返回</n-button>
          <span style="font-weight:700;color:#1e3a8a">世界设定</span>
          <n-tag v-if="mode==='adopt'" type="success" size="small">已从现有配置读取</n-tag>
          <n-tag v-else type="warning" size="small">默认值</n-tag>
          <n-button size="tiny" secondary @click="resetWorldSettings()" title="还原为游戏默认值">
            ↺ 还原默认
          </n-button>
        </n-flex>

        <n-scrollbar style="max-height:55vh">
          <div style="padding-right:8px">
            <div v-for="grp in WS_GROUPS" :key="grp.label" style="margin-bottom:18px">
              <n-divider title-placement="left" style="margin:0 0 10px">
                <span style="font-size:11px;font-weight:700;text-transform:uppercase;color:#6b7280;letter-spacing:.05em">{{ grp.label }}</span>
              </n-divider>
              <n-grid :cols="2" :x-gap="12" :y-gap="8">
                <n-gi v-for="k in grp.keys" :key="k">
                  <n-form-item :label="WS_LABELS[k]||k" label-placement="top" size="small" style="margin:0">
                    <n-switch v-if="isBool(k)"
                      :value="boolVal(k)"
                      @update:value="v=>setBool(k,v)"
                      size="small">
                      <template #checked>是</template>
                      <template #unchecked>否</template>
                    </n-switch>
                    <n-select v-else-if="isSelect(k)"
                      :value="ws[k]"
                      @update:value="v=>{ ws[k]=v }"
                      size="small"
                      :options="selectOptions[k]"
                      style="width:100%" />
                    <n-input-number v-else-if="isNumeric(k)"
                      :value="parseFloat(ws[k]) ?? numMin(k)"
                      @update:value="v=>{ ws[k]=String(v ?? numMin(k)) }"
                      size="small"
                      :step="numStep(k)"
                      :min="numMin(k)"
                      style="width:130px" />
                    <n-input v-else
                      :value="ws[k]"
                      @update:value="v=>{ ws[k]=v }"
                      size="small" />
                  </n-form-item>
                </n-gi>
              </n-grid>
            </div>
          </div>
        </n-scrollbar>

        <n-flex justify="space-between" align="center"
          style="margin-top:14px;padding-top:12px;border-top:1px solid #e5e7eb">
          <n-text depth="3" style="font-size:11px">
            保存后配置将写入 PalWorldSettings.ini，PST 将自动接入该服务器。
          </n-text>
          <n-button type="primary" :loading="worldSaving" @click="saveWorldSettings">
            保存并完成设置
          </n-button>
        </n-flex>
      </div>
    </n-card>
  </n-modal>
  <!-- ── Directory browser modal ── -->
  <n-modal v-model:show="showDirBrowser" transform-origin="center"
    style="width:min(500px,96vw)" :mask-closable="true">
    <n-card :bordered="false" style="border-radius:12px" title="选择目录">
      <template #header-extra>
        <n-flex :size="6" align="center">
          <n-button size="small" :disabled="dirLoading||!dirCurrent" @click="browserUp">← 上级</n-button>
          <n-button size="small" :disabled="dirLoading||!dirCurrent" @click="startMkdir">新建文件夹</n-button>
        </n-flex>
      </template>

      <n-text code style="font-size:12px;display:block;margin-bottom:8px;word-break:break-all">
        {{ dirCurrent || '选择驱动器' }}
      </n-text>

      <n-input-group v-if="mkdirShow" style="margin-bottom:8px">
        <n-input ref="mkdirInputEl" v-model:value="mkdirName"
          placeholder="新文件夹名称" size="small"
          @keydown.enter="confirmMkdir" @keydown.esc="mkdirShow=false" />
        <n-button size="small" type="primary" @click="confirmMkdir">确定</n-button>
        <n-button size="small" @click="mkdirShow=false">取消</n-button>
      </n-input-group>
      <n-text v-if="mkdirError" type="error"
        style="font-size:11px;display:block;margin-bottom:6px">{{ mkdirError }}</n-text>

      <div style="height:300px;overflow-y:auto;border:1px solid #e5e7eb;border-radius:6px">
        <n-spin :show="dirLoading">
          <n-text v-if="dirError" type="error"
            style="display:block;padding:24px;text-align:center">{{ dirError }}</n-text>
          <template v-else-if="!dirCurrent">
            <n-empty v-if="!dirRoots.length" description="无可用驱动器" style="padding:30px"/>
            <n-list v-else hoverable clickable>
              <n-list-item v-for="root in dirRoots" :key="root"
                style="cursor:pointer" @click="browserSelectRoot(root)">
                <n-flex align="center" :size="8">
                  <svg stroke="#2563eb" fill="none" stroke-width="2" viewBox="0 0 24 24"
                    stroke-linecap="round" stroke-linejoin="round" width="15" height="15">
                    <rect x="2" y="6" width="20" height="14" rx="2"/>
                    <path d="M2 10h20"/><line x1="6" y1="14" x2="6.01" y2="14"/>
                  </svg>
                  <strong>{{ root }}</strong>
                </n-flex>
              </n-list-item>
            </n-list>
          </template>
          <template v-else>
            <n-empty v-if="!dirEntries.length" description="无子目录" style="padding:30px"/>
            <n-list v-else hoverable clickable>
              <n-list-item v-for="e in dirEntries" :key="e.Name||e.name"
                style="cursor:pointer" @click="browserEnter(e)">
                <n-flex align="center" :size="8">
                  <svg stroke="#2563eb" fill="none" stroke-width="2" viewBox="0 0 24 24"
                    stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                  </svg>
                  {{ e.Name||e.name }}
                </n-flex>
              </n-list-item>
            </n-list>
          </template>
        </n-spin>
      </div>

      <template #footer>
        <n-flex justify="space-between" align="center">
          <n-text depth="3" style="font-size:11px">已选：{{ dirCurrent||"(未选择)" }}</n-text>
          <n-flex :size="8">
            <n-button @click="showDirBrowser=false">取消</n-button>
            <n-button type="primary" :disabled="!dirCurrent" @click="browserConfirm">
              选择此目录
            </n-button>
          </n-flex>
        </n-flex>
      </template>
    </n-card>
  </n-modal>
</template>

<style scoped>
.sdot {
  width: 22px; height: 22px; border-radius: 50%;
  border: 2px solid #d1d5db; background: #f3f4f6; color: #9ca3af;
  font-size: 11px; font-weight: 700;
  display: flex; align-items: center; justify-content: center;
}
.sdot-on { border-color: #2563eb; background: #2563eb; color: #fff; }
.sline { width: 18px; height: 2px; background: #e5e7eb; }
.choice-card {
  border: 2px solid #e5e7eb; border-radius: 10px; background: #fff;
  cursor: pointer; padding: 20px 14px;
  display: flex; flex-direction: column; align-items: center; gap: 8px;
  transition: border-color .15s, background .15s, box-shadow .15s;
  user-select: none;
}
.choice-card:hover {
  border-color: #2563eb; background: #eff6ff;
  box-shadow: 0 0 0 3px rgba(37,99,235,.1);
}
.choice-label { font-weight: 700; font-size: 14px; color: #1e3a8a; }
.choice-desc  { font-size: 11px; color: #6b7280; text-align: center; }
.install-log {
  margin-top: 10px; font-family: monospace; font-size: 11px;
  max-height: 180px; overflow-y: auto;
  background: #f3f4f6; border: 1px solid #e5e7eb;
  border-radius: 6px; padding: 8px 10px; color: #374151; line-height: 1.6;
}
</style>
