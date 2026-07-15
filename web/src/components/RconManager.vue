<script setup>
import { computed, nextTick, ref, watch } from "vue";
import dayjs from "dayjs";
import { useDialog, useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import itemMap from "@/assets/items.json";
import palMap from "@/assets/pal.json";
import {
  RCON_PLACEHOLDERS,
  extractRconPlaceholders,
  resolveRconTemplate,
} from "@/utils/rconTemplate";

const props = defineProps({ show: Boolean });
const emit  = defineEmits(["update:show"]);
const { t, locale } = useI18n();
const message = useMessage();
const dialog  = useDialog();
const api     = new ApiService();
const asArray = (v) => (Array.isArray(v) ? v : []);

const visible = computed({
  get: () => props.show,
  set: (v) => emit("update:show", v),
});

const loading   = ref(false);
const commands  = ref([]);
const tasks     = ref([]);
const players   = ref([]);
const activeTab = ref("console");

// ── placeholders ──────────────────────────────────────────────────────────────
const selectedPlayerUid = ref(null);
const selectedItem      = ref(null);
const selectedPal       = ref(null);
const selectedPlayer    = computed(() =>
  players.value.find((p) => p.player_uid === selectedPlayerUid.value)
);
const playerOptions = computed(() =>
  players.value.map((p) => ({ label: `${p.nickname} (${p.player_uid})`, value: p.player_uid }))
);
const itemOptions = computed(() =>
  (itemMap[locale.value] || itemMap.zh).map((i) => ({ label: `${i.name} · ${i.key}`, value: i.key }))
);
const palOptions = computed(() =>
  Object.entries(palMap[locale.value] || palMap.zh).map(([k, v]) => ({ label: `${v} · ${k}`, value: k }))
);
const commandOptions = computed(() =>
  commands.value.map((c) => ({ label: c.remark || c.command, value: c.uuid }))
);
const uploadHeaders = computed(() => ({
  Authorization: `Bearer ${localStorage.getItem("palworld_token") || ""}`,
}));
const placeholderLabels = computed(() =>
  RCON_PLACEHOLDERS.map((p) => ({ ...p, token: `{${p.key}}`, label: t(`rconManager.placeholder.${p.key}`) }))
);

// ── load ──────────────────────────────────────────────────────────────────────
const commandContent = ref({});
const loadData = async () => {
  loading.value = true;
  try {
    const [cr, tr, pr] = await Promise.all([
      api.getRconCommands(),
      api.getRconTasks(),
      api.getPlayerList({ order_by: "last_online", desc: true }),
    ]);
    commands.value = asArray(cr.data.value);
    tasks.value    = asArray(tr.data.value);
    players.value  = asArray(pr.data.value);
    for (const c of commands.value) {
      if (commandContent.value[c.uuid] === undefined) commandContent.value[c.uuid] = "";
    }
  } finally {
    loading.value = false;
  }
};
watch(() => props.show, (v) => { if (v) loadData(); });

// ── built-in command catalogue ────────────────────────────────────────────────
const BUILTIN_COMMANDS = [
  { group: "服务器", cmds: [
    { cmd: "Info",                       desc: "显示服务器信息" },
    { cmd: "Save",                       desc: "保存世界数据" },
    { cmd: "Broadcast {消息}",            desc: "广播消息（不支持空格）" },
    { cmd: "Shutdown {秒} {消息}",        desc: "倒数后关闭服务器", danger: true },
    { cmd: "DoExit",                     desc: "立即强制停止服务器", danger: true },
    { cmd: "version",                    desc: "显示游戏与 PalDefender 版本" },
    { cmd: "reloadcfg",                  desc: "重新加载配置与封禁名单" },
    { cmd: "pgbroadcast {消息}",          desc: "广播消息（支持空格）" },
    { cmd: "alert {消息}",               desc: "发送醒目警示消息" },
  ]},
  { group: "玩家", cmds: [
    { cmd: "ShowPlayers",                desc: "列出所有在线玩家" },
    { cmd: "getpos {玩家}",              desc: "查询玩家坐标" },
    { cmd: "renameplayer {旧名} {新名}", desc: "重命名玩家" },
    { cmd: "give_exp {玩家} {数量}",     desc: "给予经验值" },
    { cmd: "givestats {玩家} {属性} {n}",desc: "给予状态点数" },
  ]},
  { group: "管理与封禁", cmds: [
    { cmd: "KickPlayer {SteamID}",       desc: "踢出玩家" },
    { cmd: "BanPlayer {SteamID}",        desc: "封禁玩家", danger: true },
    { cmd: "UnBanPlayer {SteamID}",      desc: "解除封禁" },
    { cmd: "setadmin {玩家}",            desc: "授予/取消管理员权限" },
    { cmd: "addadminip {IP}",            desc: "将 IP 加入管理员白名单" },
    { cmd: "getip {玩家}",               desc: "查询玩家 IP" },
    { cmd: "kick {玩家} {原因}",         desc: "踢出玩家（可附原因）" },
    { cmd: "ban {玩家} {原因}",          desc: "封禁玩家（可附原因）", danger: true },
    { cmd: "ipban {玩家}",               desc: "IP 封禁玩家", danger: true },
    { cmd: "unban {UserId}",             desc: "解除封禁 UserId" },
    { cmd: "banip {IP}",                 desc: "封禁 IP", danger: true },
    { cmd: "unbanip {IP}",               desc: "解除 IP 封禁" },
    { cmd: "whitelist_add {玩家}",       desc: "加入白名单" },
    { cmd: "whitelist_remove {玩家}",    desc: "移出白名单" },
    { cmd: "whitelist_get",              desc: "列出白名单" },
  ]},
  { group: "世界", cmds: [
    { cmd: "settime {小时}",             desc: "设定世界时间" },
  ]},
  { group: "道具", cmds: [
    { cmd: "give {玩家} {道具ID} {n}",   desc: "给予道具" },
    { cmd: "delitem {玩家} {道具ID} {n}",desc: "移除道具" },
    { cmd: "clearinv {玩家}",            desc: "清空玩家背包", danger: true },
    { cmd: "give_relic {玩家} {n}",      desc: "给予灵魂雕像" },
  ]},
  { group: "帕鲁", cmds: [
    { cmd: "givepal {玩家} {PalID}",     desc: "给予帕鲁" },
    { cmd: "spawnpal {PalID}",           desc: "生成野生帕鲁" },
    { cmd: "giveegg {玩家} {PalID}",     desc: "给予帕鲁蛋" },
    { cmd: "exportpals {玩家}",          desc: "导出玩家的帕鲁" },
    { cmd: "deletepals {条件}",          desc: "依条件删除帕鲁", danger: true },
    { cmd: "getskinids",                 desc: "列出所有帕鲁造型 ID" },
  ]},
  { group: "科技", cmds: [
    { cmd: "learntech {ID|all}",         desc: "解锁科技（all = 全部）" },
    { cmd: "unlearntech {ID|all}",       desc: "锁回科技（all = 全部）" },
    { cmd: "givetechpoints {玩家} {n}",  desc: "给予科技点数" },
    { cmd: "givebosstechpoints {玩家} {n}",desc: "给予古代科技点数" },
    { cmd: "gettechids",                 desc: "列出所有科技 ID" },
  ]},
  { group: "据点与公会", cmds: [
    { cmd: "getnearestbase",             desc: "查询最近的据点拥有者" },
    { cmd: "killnearestbase",            desc: "摧毁最近的据点", danger: true },
    { cmd: "setguildleader {玩家}",      desc: "指定公会会长" },
    { cmd: "exportguilds",               desc: "导出所有公会为 JSON" },
  ]},
];

const PINNED_CMDS = [
  { cmd: "givepal_j",  desc: "自定义帕鲁（词条/体质/星星）",       star: true },
  { cmd: "giveegg_j",  desc: "自定义帕鲁蛋（词条/体质/星星）",     star: true },
  { cmd: "giveitems",  desc: "批量给予道具（菜单+数量）",           star: true },
  { cmd: "tp",         desc: "传送玩家（玩家/地图坐标）",           star: true },
];

// ── console tab ───────────────────────────────────────────────────────────────
const cmdSearch    = ref("");
const consoleInput = ref("");
const consoleLog   = ref([]);
const executing    = ref(false);
const logEl        = ref(null);

const totalCount = computed(() =>
  PINNED_CMDS.length + BUILTIN_COMMANDS.reduce((s, g) => s + g.cmds.length, 0)
);

const filteredPinned = computed(() => {
  const q = cmdSearch.value.trim().toLowerCase();
  if (!q) return PINNED_CMDS;
  return PINNED_CMDS.filter((c) => c.cmd.toLowerCase().includes(q) || c.desc.includes(q));
});

const filteredGroups = computed(() => {
  const q = cmdSearch.value.trim().toLowerCase();
  const src = BUILTIN_COMMANDS;
  if (!q) return src;
  return src
    .map((g) => ({ ...g, cmds: g.cmds.filter((c) => c.cmd.toLowerCase().includes(q) || c.desc.includes(q)) }))
    .filter((g) => g.cmds.length > 0);
});

// Current command hint shown below input
const cmdHint = ref('');

function selectCmd(cmdStr, descStr) {
  consoleInput.value = cmdStr.replace(/\{[^}]*\}/g, '').trim() + ' ';
  // Build usage: replace {param} with <param> style
  const usage = cmdStr.replace(/\{([^}]*)\}/g, '<$1>');
  cmdHint.value = (descStr ? `${descStr}  ·  ` : '') + `用法: ${usage}`;
  onInputChange();
  nextTick(() => {
    const el = document.getElementById('rcon-console-input');
    if (el) { el.focus(); el.setSelectionRange(el.value.length, el.value.length); }
  });
}

// Build a lookup: itemKey -> zhName from items.json
const itemZhMap = (() => {
  const map = {};
  const list = (itemMap['zh'] || itemMap['en'] || []);
  for (const item of list) {
    if (item.key) map[item.key.toLowerCase()] = { zh: item.name, en: (itemMap['en'] || []).find(e => e.key === item.key)?.name || item.key };
  }
  return map;
})();

function formatReply(text) {
  if (typeof text !== 'string') return text;
  const trimmed = text.trim();
  // JSON array of strings → item list table
  if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
    try {
      const arr = JSON.parse(trimmed);
      if (Array.isArray(arr) && arr.length > 0 && typeof arr[0] === 'string') {
        return arr.map(k => {
          const info = itemZhMap[k.toLowerCase()];
          return { key: k, zh: info?.zh || k, en: info?.en || k };
        });
      }
    } catch { /* not JSON, fall through */ }
  }
  // JSON object → formatted key-value table
  if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
    try {
      const obj = JSON.parse(trimmed);
      if (obj && typeof obj === 'object' && !Array.isArray(obj)) {
        return { __type: 'object', data: obj };
      }
    } catch { /* not JSON, fall through */ }
  }
  return text;
}

async function runConsole() {
  const cmd = consoleInput.value.trim();
  if (!cmd || executing.value) return;
  executing.value = true;
  // Clear previous output before each new command
  consoleLog.value = [];
  consoleLog.value.push({ type: "cmd", text: cmd, time: dayjs().format("HH:mm:ss") });
  try {
    const { data, statusCode } = await api.execRconCommand(cmd);
    const rawReply = data.value?.message || data.value?.result
      || (statusCode.value === 200 ? "✓ 执行成功（无输出）" : "执行失败");
    // Try to parse as JSON array of strings and render as item list
    const formattedReply = formatReply(rawReply);
    consoleLog.value.push({
      type: statusCode.value === 200 ? "ok" : "err",
      text: formattedReply,
      isHtml: Array.isArray(formattedReply),
      time: dayjs().format("HH:mm:ss"),
    });
    if (statusCode.value === 200) consoleInput.value = "";
  } catch (e) {
    consoleLog.value.push({ type: "err", text: String(e), time: dayjs().format("HH:mm:ss") });
  } finally {
    executing.value = false;
    nextTick(() => {
      if (logEl.value?.$el) logEl.value.$el.scrollTop = logEl.value.$el.scrollHeight;
    });
  }
}

// ── Autocomplete state ──────────────────────────────────────────────────
const acList      = ref([]);  // current suggestion list
const acIndex     = ref(-1); // selected index (-1 = none)
const acShow      = ref(false);

// Build a flat list of all known param values for the current cmd token
function buildAcList(input) {
  const tokens = input.split(/\s+/);
  const cmd = tokens[0].toLowerCase();
  const partialLast = tokens[tokens.length - 1].toLowerCase();

  // For give/delitem/givepal/giveegg commands, suggest item/pal keys
  const itemCmds = ['give','delitem','giveitem','giveitems'];
  const palCmds  = ['givepal','giveegg','givepal_j','giveegg_j','spawnpal'];
  const playerCmds = ['kick','ban','ipban','getpos','setguildleader','getip','renameplayer',
    'give','delitem','givepal','giveegg','give_exp','givetechpoints','givebosstechpoints',
    'giveitem','giveitems','tp','exportpals','clearinv','give_relic'];
  const techCmds = ['learntech','unlearntech'];

  let pool = [];

  // First token = command itself
  if (tokens.length === 1) {
    const allCmds = [
      ...PINNED_CMDS.map(c => c.cmd.split(' ')[0]),
      ...BUILTIN_COMMANDS.flatMap(g => g.cmds.map(c => c.cmd.split(' ')[0])),
    ];
    pool = [...new Set(allCmds)];
  } else if (palCmds.some(x => cmd.startsWith(x))) {
    pool = palOptions.value.map(o => o.value);
  } else if (itemCmds.some(x => cmd.startsWith(x))) {
    pool = itemOptions.value.map(o => o.value);
  } else if (techCmds.some(x => cmd.startsWith(x))) {
    pool = ['all', ...Array.from(new Set(itemOptions.value.map(o => o.value)))];
  } else if (playerCmds.some(x => cmd.startsWith(x)) && tokens.length <= 2) {
    pool = players.value.map(p => p.nickname);
  }

  if (!pool.length) return [];
  const filtered = partialLast
    ? pool.filter(v => v.toLowerCase().startsWith(partialLast))
    : pool.slice(0, 12);
  return filtered.slice(0, 18);
}

function applyAcSuggestion(suggestion) {
  const tokens = consoleInput.value.split(/\s+/);
  tokens[tokens.length - 1] = suggestion;
  consoleInput.value = tokens.join(' ') + ' ';
  acShow.value = false;
  acList.value = [];
  acIndex.value = -1;
  nextTick(() => {
    const el = document.getElementById('rcon-console-input');
    if (el) { el.focus(); el.setSelectionRange(el.value.length, el.value.length); }
  });
}

function onInputChange() {
  acList.value = buildAcList(consoleInput.value);
  acShow.value = acList.value.length > 0;
  acIndex.value = -1;
}

function onInputKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault();
    if (acShow.value && acIndex.value >= 0) {
      applyAcSuggestion(acList.value[acIndex.value]);
    } else {
      acShow.value = false;
      runConsole();
    }
    return;
  }
  if (e.key === 'Tab') {
    e.preventDefault();
    if (!acShow.value) {
      acList.value = buildAcList(consoleInput.value);
      acShow.value = acList.value.length > 0;
      acIndex.value = acList.value.length > 0 ? 0 : -1;
    } else {
      acIndex.value = (acIndex.value + 1) % acList.value.length;
    }
    if (acIndex.value >= 0) applyAcSuggestion(acList.value[acIndex.value]);
    return;
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault();
    if (acShow.value && acList.value.length) {
      acIndex.value = acIndex.value <= 0 ? acList.value.length - 1 : acIndex.value - 1;
    }
    return;
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault();
    if (acShow.value && acList.value.length) {
      acIndex.value = (acIndex.value + 1) % acList.value.length;
    }
    return;
  }
  if (e.key === 'Escape') { acShow.value = false; acIndex.value = -1; }
}

function onInputWheel(e) {
  if (!acShow.value || !acList.value.length) return;
  e.preventDefault();
  if (e.deltaY > 0) {
    acIndex.value = (acIndex.value + 1) % acList.value.length;
  } else {
    acIndex.value = acIndex.value <= 0 ? acList.value.length - 1 : acIndex.value - 1;
  }
}

// ── custom commands tab ───────────────────────────────────────────────────────
const commandModal       = ref(false);
const editingCommandUUID = ref(null);
const commandForm        = ref({ command: "", remark: "", placeholder: "" });

const resolveTemplate = (tmpl) =>
  resolveRconTemplate(tmpl, {
    player: selectedPlayer.value,
    item:   selectedItem.value,
    pal:    selectedPal.value,
  });

const fillCommand = (command) => {
  const r = resolveTemplate(command.placeholder || "");
  commandContent.value[command.uuid] = r.content;
  if (r.missing.length)
    message.warning(t("rconManager.missingSelection", { fields: r.missing.map((k) => t(`rconManager.placeholder.${k}`)).join(", ") }));
  else if (r.unknown.length)
    message.warning(t("rconManager.unknownPlaceholder", { fields: r.unknown.join(", ") }));
  else
    message.success(t("rconManager.filled"));
};

const executeCommand = async (command) => {
  const { data, statusCode } = await api.sendRconCommand({
    uuid: command.uuid,
    content: commandContent.value[command.uuid] || "",
  });
  if (statusCode.value === 200) message.success(data.value?.message || t("rconManager.executed"));
  else message.error(data.value?.error || t("rconManager.executeFailed"));
};

const openCommandModal = (cmd = null) => {
  editingCommandUUID.value = cmd?.uuid || null;
  commandForm.value = cmd
    ? { command: cmd.command, remark: cmd.remark, placeholder: cmd.placeholder }
    : { command: "", remark: "", placeholder: "" };
  commandModal.value = true;
};

const saveCommand = async () => {
  if (!commandForm.value.command.trim() || !commandForm.value.remark.trim()) {
    message.warning(t("rconManager.commandRequired")); return;
  }
  const r = editingCommandUUID.value
    ? await api.putRconCommand(editingCommandUUID.value, commandForm.value)
    : await api.addRconCommand(commandForm.value);
  if (r.statusCode.value === 200) {
    message.success(t("rconManager.commandSaved"));
    commandModal.value = false;
    await loadData();
  } else {
    message.error(r.data.value?.error || t("rconManager.saveFailed"));
  }
};

const removeCommand = (cmd) => {
  dialog.warning({
    title: t("message.warn"),
    content: t("rconManager.removeCommandConfirm", { name: cmd.remark || cmd.command }),
    positiveText: t("button.confirm"), negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      const { data, statusCode } = await api.removeRconCommand(cmd.uuid);
      if (statusCode.value === 200) { message.success(t("rconManager.commandRemoved")); await loadData(); }
      else message.error(data.value?.error || t("rconManager.removeFailed"));
    },
  });
};

const importFinished = async () => {
  message.success(t("message.importRconSuccess"));
  commandModal.value = false;
  await loadData();
};
const importFailed = () => message.error(t("message.importRconFail", { err: "" }));

watch([selectedPlayerUid, selectedItem, selectedPal], () => {
  for (const c of commands.value) {
    if (c.placeholder) commandContent.value[c.uuid] = resolveTemplate(c.placeholder).content;
  }
  if (taskModal.value && taskForm.value.rcon_uuid) selectTaskCommand(taskForm.value.rcon_uuid);
});

// ── tasks tab ─────────────────────────────────────────────────────────────────
const PRESET_TASKS = [
  { name: '定时保存世界',    cmd: 'Save',                    content: '', intervalMin: 30,  cron: '*/30 * * * *' },
  { name: '定时广播提醒',    cmd: 'Broadcast ServerRunning', content: '', intervalMin: 60,  cron: '0 * * * *' },
  { name: '定时重启服务器',  cmd: 'Shutdown 60 AutoRestart', content: '', intervalMin: 480, cron: '0 4 * * *' },
  { name: '提示',    cmd: 'Broadcast BackupStarting',content: '', intervalMin: 1440,cron: '0 3 * * *' },
];

const taskModal       = ref(false);
const editingTaskUUID = ref(null);
const scheduleType    = ref("interval");
const intervalMinutes = ref(15);
const dailyTime       = ref("04:00");
const weeklyDay       = ref(1);
const weeklyTime      = ref("04:00");
const customCron      = ref("0 4 * * *");
const taskForm        = ref({ name: '', rcon_uuid: null, inlineCmd: '', content: '', cron: '', enabled: true });
const taskCmdMode     = ref('custom');  
const scheduleTypeOptions = computed(() => [
  { label: t("rconManager.schedule.interval"), value: "interval" },
  { label: t("rconManager.schedule.daily"),    value: "daily"    },
  { label: t("rconManager.schedule.weekly"),   value: "weekly"   },
  { label: t("rconManager.schedule.custom"),   value: "custom"   },
]);
const weekOptions = computed(() =>
  Array.from({ length: 7 }, (_, i) => ({ value: i, label: t(`rconManager.weekday.${i}`) }))
);

const parseTime = (v) => {
  const [h, m] = (v || "00:00").split(":").map(Number);
  return { hour: Number.isFinite(h) ? h : 0, minute: Number.isFinite(m) ? m : 0 };
};
const buildCron = () => {
  if (scheduleType.value === "interval") return `*/${Math.max(1, Number(intervalMinutes.value) || 1)} * * * *`;
  if (scheduleType.value === "daily")  { const { hour, minute } = parseTime(dailyTime.value);  return `${minute} ${hour} * * *`; }
  if (scheduleType.value === "weekly") { const { hour, minute } = parseTime(weeklyTime.value); return `${minute} ${hour} * * ${weeklyDay.value}`; }
  return customCron.value.trim();
};

const selectTaskCommand = (uuid) => {
  const c = commands.value.find((x) => x.uuid === uuid);
  if (!c) return;
  taskForm.value.content = resolveTemplate(c.placeholder || "").content;
  if (!taskForm.value.name) taskForm.value.name = c.remark || c.command;
};

const parseCronIntoForm = (cron) => {
  const iM = cron.match(/^\*\/(\d+) \* \* \* \*$/);
  const wM = cron.match(/^(\d+) (\d+) \* \* ([0-6])$/);
  const dM = cron.match(/^(\d+) (\d+) \* \* \*$/);
  if (iM)      { scheduleType.value = 'interval'; intervalMinutes.value = Number(iM[1]); }
  else if (wM) { scheduleType.value = 'weekly';   weeklyTime.value = `${String(wM[2]).padStart(2,'0')}:${String(wM[1]).padStart(2,'0')}`; weeklyDay.value = Number(wM[3]); }
  else if (dM) { scheduleType.value = 'daily';    dailyTime.value  = `${String(dM[2]).padStart(2,'0')}:${String(dM[1]).padStart(2,'0')}`; }
  else         { scheduleType.value = 'custom';   customCron.value = cron; }
};

const openTaskModal = (cmd = null, existing = null, preset = null) => {
  editingTaskUUID.value = existing?.uuid || null;
  if (existing) {
    const hasCustom = !!existing.rcon_uuid;
    taskCmdMode.value = hasCustom ? 'custom' : 'inline';
    taskForm.value = { name: existing.name, rcon_uuid: existing.rcon_uuid || null,
      inlineCmd: existing.inline_cmd || '', content: existing.content, cron: existing.cron, enabled: existing.enabled };
    parseCronIntoForm(existing.cron);
  } else if (preset) {
    taskCmdMode.value = 'inline';
    taskForm.value = { name: preset.name, rcon_uuid: null, inlineCmd: preset.cmd, content: preset.content, cron: '', enabled: true };
    parseCronIntoForm(preset.cron);
  } else {
    taskCmdMode.value = commands.value.length ? 'custom' : 'inline';
    taskForm.value = { name: cmd?.remark || cmd?.command || '', rcon_uuid: cmd?.uuid || null,
      inlineCmd: '', content: '', cron: '', enabled: true };
    scheduleType.value = 'interval'; intervalMinutes.value = 15;
    if (cmd) selectTaskCommand(cmd.uuid);
  }
  taskModal.value = true;
};

const saveTask = async () => {
  const cron = buildCron();
  const isInline = taskCmdMode.value === 'inline';
  const hasCmd = isInline ? !!taskForm.value.inlineCmd.trim() : !!taskForm.value.rcon_uuid;
  if (!taskForm.value.name.trim() || !hasCmd || !cron) {
    message.warning(t('rconManager.taskRequired')); return;
  }
  const unresolved = extractRconPlaceholders(taskForm.value.content);
  if (unresolved.length) {
    message.warning(t('rconManager.unresolvedTaskPlaceholder', { fields: unresolved.map((k) => `{${k}}`).join(', ') })); return;
  }
  let payload = { ...taskForm.value, cron };
  if (isInline) {
        let existing = commands.value.find(x => x.command === taskForm.value.inlineCmd.trim());
    if (!existing) {
      const cr = await api.addRconCommand({ command: taskForm.value.inlineCmd.trim(), remark: taskForm.value.name, placeholder: '' });
      if (cr.statusCode.value !== 200) { message.error(cr.data.value?.error || t('rconManager.saveFailed')); return; }
      await loadData();
      existing = commands.value.find(x => x.command === taskForm.value.inlineCmd.trim());
    }
    if (!existing) { message.error('无法创建关联命令'); return; }
    payload.rcon_uuid = existing.uuid;
  }
  const r = editingTaskUUID.value
    ? await api.putRconTask(editingTaskUUID.value, payload)
    : await api.addRconTask(payload);
  if (r.statusCode.value === 200) {
    message.success(t('rconManager.taskSaved'));
    taskModal.value = false;
    activeTab.value = 'tasks';
    await loadData();
  } else {
    message.error(r.data.value?.error || t('rconManager.saveFailed'));
  }
};

const toggleTask = async (task, enabled) => {
  const r = await api.putRconTask(task.uuid, { ...task, enabled });
  if (r.statusCode.value === 200) {
    message.success(t(enabled ? "rconManager.taskEnabled" : "rconManager.taskPaused"));
    await loadData();
  } else {
    message.error(r.data.value?.error || t("rconManager.saveFailed"));
  }
};

const runTask = (task) => {
  dialog.warning({
    title: t("rconManager.runNow"), content: t("rconManager.runConfirm", { name: task.name }),
    positiveText: t("button.execute"), negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      const { data, statusCode } = await api.runRconTask(task.uuid);
      if (statusCode.value === 200) { message.success(t("rconManager.executed")); await loadData(); }
      else message.error(data.value?.error || t("rconManager.executeFailed"));
    },
  });
};

const removeTask = (task) => {
  dialog.warning({
    title: t("message.warn"), content: t("rconManager.removeTaskConfirm", { name: task.name }),
    positiveText: t("button.confirm"), negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      const { data, statusCode } = await api.removeRconTask(task.uuid);
      if (statusCode.value === 200) { message.success(t("rconManager.taskRemoved")); await loadData(); }
      else message.error(data.value?.error || t("rconManager.removeFailed"));
    },
  });
};

const statusType  = (s) => s === "success" ? "success" : s === "failed" ? "error" : "default";
const formatTime  = (v) => (v ? dayjs(v).format("YYYY-MM-DD HH:mm:ss") : "—");
const drawerWidth = computed(() => Math.min(900, window.innerWidth));
</script>

<template>
  <n-drawer v-model:show="visible" :width="drawerWidth" placement="right">
    <n-drawer-content closable>
      <template #header>
        <n-flex align="center" :size="8">
          <svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24"
            stroke-linecap="round" stroke-linejoin="round" width="18" height="18">
            <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
          </svg>
          <span style="font-weight:800">指令台</span>
          <n-tag size="small" round style="font-size:11px">{{ totalCount }} 个可用指令</n-tag>
        </n-flex>
      </template>
      <n-spin :show="loading">
        <n-tabs v-model:value="activeTab" type="segment" size="small" class="mb-3">
          <n-tab name="console">指令台</n-tab>
          <n-tab name="custom">自定义命令</n-tab>
          <n-tab name="tasks">定时任务</n-tab>
        </n-tabs>

        <!-- console tab -->
        <div v-show="activeTab === 'console'" class="console-layout">
          <!-- left: command list -->
          <div class="cmd-panel">
            <div class="cmd-search-wrap">
              <svg class="cmd-search-icon" stroke="currentColor" fill="none" stroke-width="2"
                viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" width="13" height="13">
                <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
              </svg>
              <input v-model="cmdSearch" class="cmd-search-input" placeholder="搜索指令…" />
            </div>
            <div class="cmd-list">
              <template v-if="filteredPinned.length">
                <div class="cmd-group-label">⭐ GM 专属</div>
                <button v-for="c in filteredPinned" :key="c.cmd" class="cmd-item" @click="selectCmd(c.cmd, c.desc)">
                  <span class="cmd-name cmd-name--star">{{ c.cmd }}</span>
                  <span class="cmd-desc">{{ c.desc }}</span>
                </button>
              </template>
              <template v-for="g in filteredGroups" :key="g.group">
                <div class="cmd-group-label">{{ g.group }}</div>
                <button v-for="c in g.cmds" :key="c.cmd" class="cmd-item" @click="selectCmd(c.cmd, c.desc)">
                  <span class="cmd-name" :class="{ 'cmd-name--danger': c.danger }">
                    {{ c.cmd }}
                    <span v-if="c.danger" class="cmd-danger-tag">危险</span>
                  </span>
                  <span class="cmd-desc">{{ c.desc }}</span>
                </button>
              </template>
              <div v-if="!filteredPinned.length && !filteredGroups?.length" class="cmd-empty">没有匹配的指令</div>
            </div>
          </div>

          <!-- right: terminal -->
          <div class="terminal-panel">
            <n-card size="small" style="flex-shrink:0">
              <n-grid cols="3" :x-gap="8" :y-gap="6">
                <n-gi>
                  <n-select v-model:value="selectedPlayerUid" filterable clearable size="small"
                    :options="playerOptions" :placeholder="$t('input.selectPlayer')" />
                </n-gi>
                <n-gi>
                  <n-select v-model:value="selectedItem" filterable clearable size="small"
                    :options="itemOptions" :placeholder="$t('input.selectItem')" />
                </n-gi>
                <n-gi>
                  <n-select v-model:value="selectedPal" filterable clearable size="small"
                    :options="palOptions" :placeholder="$t('input.selectPal')" />
                </n-gi>
              </n-grid>
            </n-card>

            <n-scrollbar ref="logEl" class="terminal-log">
              <div v-if="!consoleLog.length" class="terminal-placeholder">
                点击左侧指令快速填入，或直接输入任意 RCON 指令后按 Enter 执行
              </div>
              <div v-for="(entry, i) in consoleLog" :key="i"
                :class="['terminal-entry', `terminal-entry--${entry.type}`]">
                <span class="terminal-time">{{ entry.time }}</span>
                <span class="terminal-prompt"
                  :class="entry.type === 'ok' ? 'terminal-prompt--ok' : entry.type === 'err' ? 'terminal-prompt--err' : ''">
                  {{ entry.type === 'cmd' ? '›' : entry.type === 'ok' ? '✓' : '✗' }}
                </span>
                <!-- Array reply: render as EN/CN two-column list -->
                <div v-if="entry.isHtml && Array.isArray(entry.text)" class="terminal-item-list">
                  <div class="til-header">
                    <span class="til-col til-en">英文 ID</span>
                    <span class="til-col til-zh">中文名称</span>
                  </div>
                  <div v-for="item in entry.text" :key="item.key" class="til-row"
                    @click="consoleInput += item.key + ' '" style="cursor:pointer">
                    <span class="til-col til-en">{{ item.key }}</span>
                    <span class="til-col til-zh">{{ item.zh !== item.key ? item.zh : '—' }}</span>
                  </div>
                </div>
                <!-- JSON object → key-value table -->
                <div v-else-if="entry.text && entry.text.__type === 'object'" class="terminal-obj">
                  <div v-for="(val, key) in entry.text.data" :key="key" class="terminal-obj-row">
                    <span class="terminal-obj-key">{{ key }}</span>
                    <span class="terminal-obj-val">
                      <template v-if="val && typeof val === 'object'">
                        <span v-for="(v2, k2) in val" :key="k2" class="terminal-obj-sub">
                          <span class="terminal-obj-subkey">{{ k2 }}</span>: {{ v2 }}
                        </span>
                      </template>
                      <template v-else>{{ val }}</template>
                    </span>
                  </div>
                </div>
                <span v-else class="terminal-text">{{ entry.text }}</span>
              </div>
            </n-scrollbar>

            <div class="terminal-input-wrap">
              <!-- Autocomplete dropdown -->
              <div v-if="acShow && acList.length" class="ac-dropdown">
                <div v-for="(sug, idx) in acList" :key="sug"
                  :class="['ac-item', idx === acIndex ? 'ac-item--active' : '']"
                  @mousedown.prevent="applyAcSuggestion(sug)">
                  {{ sug }}
                </div>
              </div>
              <div class="terminal-input-bar">
                <svg class="terminal-input-icon" stroke="currentColor" fill="none" stroke-width="2"
                  viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                  <polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/>
                </svg>
                <input id="rcon-console-input" v-model="consoleInput" class="terminal-input"
                  placeholder="输入 RCON 指令，例如 ShowPlayers  |  Tab 补全  |  Esc 关闭提示"
                  autocomplete="off" spellcheck="false"
                  @input="onInputChange"
                  @keydown="onInputKeydown"
                  @wheel.passive="onInputWheel"
                  @blur="setTimeout(()=>{ acShow=false }, 150)" />
                <n-button type="primary" size="small" class="terminal-send-btn"
                  :loading="executing" :disabled="!consoleInput.trim()" @click="runConsole">
                  <template #icon>
                    <svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24"
                      stroke-linecap="round" stroke-linejoin="round" width="12" height="12">
                      <polygon points="5 3 19 12 5 21 5 3"/>
                    </svg>
                  </template>
                  执行
                </n-button>
              </div>
            </div>
            <div v-if="cmdHint" class="cmd-hint-bar">{{ cmdHint }}</div>
          </div>
        </div>

        <!-- custom commands tab -->
        <div v-show="activeTab === 'custom'">
          <n-card size="small" :title="$t('rconManager.materialTitle')" class="mb-3">
            <n-grid cols="1 700:3" :x-gap="12" :y-gap="8">
              <n-gi><n-select v-model:value="selectedPlayerUid" filterable clearable :options="playerOptions" :placeholder="$t('input.selectPlayer')" /></n-gi>
              <n-gi><n-select v-model:value="selectedItem" filterable clearable :options="itemOptions" :placeholder="$t('input.selectItem')" /></n-gi>
              <n-gi><n-select v-model:value="selectedPal" filterable clearable :options="palOptions" :placeholder="$t('input.selectPal')" /></n-gi>
            </n-grid>
            <n-flex class="mt-2" align="center" wrap>
              <n-text depth="3" style="font-size:11px">{{ $t("rconManager.placeholderHelp") }}</n-text>
              <n-tooltip v-for="p in placeholderLabels" :key="p.key">
                <template #trigger><n-tag size="small" round>{{ p.token }}</n-tag></template>
                {{ p.label }}
              </n-tooltip>
            </n-flex>
          </n-card>
          <n-flex justify="space-between" align="center" class="mb-3">
            <n-text depth="3">{{ $t("rconManager.commandDesc") }}</n-text>
            <n-button type="primary" @click="openCommandModal()">{{ $t("button.addRcon") }}</n-button>
          </n-flex>
          <n-empty v-if="commands.length === 0" :description="$t('rconManager.noCommands')" />
          <n-space v-else vertical size="medium">
            <n-card v-for="command in commands" :key="command.uuid" size="small">
              <template #header>
                <n-flex align="center">
                  <n-text strong>{{ command.remark || command.command }}</n-text>
                  <n-code :code="command.command" language="shell" inline />
                </n-flex>
              </template>
              <template #header-extra>
                <n-button text type="primary" @click="openCommandModal(command)">{{ $t("rconManager.edit") }}</n-button>
              </template>
              <n-text depth="3" class="block mb-2">{{ $t("rconManager.argumentTemplate") }}: <n-code :code="command.placeholder || '—'" inline /></n-text>
              <n-input v-model:value="commandContent[command.uuid]" type="textarea" autosize :placeholder="$t('rconManager.commandContent')" />
              <n-flex class="mt-3" justify="end">
                <n-button @click="fillCommand(command)">{{ $t("button.fill") }}</n-button>
                <n-button @click="openTaskModal(command)">{{ $t("rconManager.addTask") }}</n-button>
                <n-button type="primary" @click="executeCommand(command)">{{ $t("button.execute") }}</n-button>
                <n-button type="error" secondary @click="removeCommand(command)">{{ $t("button.remove") }}</n-button>
              </n-flex>
            </n-card>
          </n-space>
        </div>

        <!-- tasks tab -->
        <div v-show="activeTab === 'tasks'">
          <!-- preset quick-create buttons -->
          <n-card size="small" class="mb-3" title="快速创建常用任务">
            <n-flex wrap :size="8">
              <n-button v-for="p in PRESET_TASKS" :key="p.name" size="small" secondary
                @click="openTaskModal(null, null, p)">
                {{ p.name }}
              </n-button>
            </n-flex>
          </n-card>
          <n-flex justify="space-between" align="center" class="mb-3">
            <n-text depth="3">{{ $t("rconManager.taskDesc") }}</n-text>
            <n-button type="primary" @click="openTaskModal()">{{ $t("rconManager.addTask") }}</n-button>
          </n-flex>
          <n-empty v-if="tasks.length === 0" :description="$t('rconManager.noTasks')" />
          <n-space v-else vertical size="medium">
            <n-card v-for="task in tasks" :key="task.uuid" size="small">
              <template #header>
                <n-flex align="center">
                  <n-text strong>{{ task.name }}</n-text>
                  <n-tag size="small" :type="statusType(task.last_status)">{{ $t(`rconManager.status.${task.last_status || 'never'}`) }}</n-tag>
                </n-flex>
              </template>
              <template #header-extra>
                <n-switch :value="task.enabled" @update:value="(v) => toggleTask(task, v)" />
              </template>
              <n-descriptions label-placement="left" :column="1" size="small">
                <n-descriptions-item :label="$t('rconManager.boundCommand')">{{ task.rcon_remark || task.rcon_uuid }}</n-descriptions-item>
                <n-descriptions-item :label="$t('rconManager.argumentContent')"><n-code :code="task.content || '—'" inline /></n-descriptions-item>
                <n-descriptions-item :label="$t('rconManager.scheduleLabel')"><n-code :code="task.cron" inline /></n-descriptions-item>
                <n-descriptions-item :label="$t('rconManager.nextRun')">{{ task.enabled ? formatTime(task.next_run_at) : $t("rconManager.paused") }}</n-descriptions-item>
                <n-descriptions-item :label="$t('rconManager.lastRun')">{{ formatTime(task.last_run_at) }} · {{ $t("rconManager.runCount", { count: task.run_count }) }}</n-descriptions-item>
                <n-descriptions-item v-if="task.last_result" :label="$t('rconManager.lastResult')">{{ task.last_result }}</n-descriptions-item>
                <n-descriptions-item v-if="task.last_error" :label="$t('rconManager.lastError')"><n-text type="error">{{ task.last_error }}</n-text></n-descriptions-item>
              </n-descriptions>
              <n-flex class="mt-3" justify="end">
                <n-button @click="openTaskModal(null, task)">{{ $t("rconManager.edit") }}</n-button>
                <n-button type="primary" secondary @click="runTask(task)">{{ $t("rconManager.runNow") }}</n-button>
                <n-button type="error" secondary @click="removeTask(task)">{{ $t("button.remove") }}</n-button>
              </n-flex>
            </n-card>
          </n-space>
        </div>

      </n-spin>
    </n-drawer-content>
  </n-drawer>

  <!-- edit command modal -->
  <n-modal v-model:show="commandModal" preset="card"
    :title="editingCommandUUID ? $t('rconManager.editCommand') : $t('button.addRcon')"
    style="width:min(92vw,620px)">
    <n-form label-placement="top">
      <n-form-item :label="$t('input.remark')" required>
        <n-input v-model:value="commandForm.remark" />
      </n-form-item>
      <n-form-item :label="$t('input.rcon')" required>
        <n-input v-model:value="commandForm.command" />
      </n-form-item>
      <n-form-item :label="$t('rconManager.argumentTemplate')">
        <n-input v-model:value="commandForm.placeholder" type="textarea" :placeholder="$t('rconManager.templateExample')" />
      </n-form-item>
    </n-form>
    <n-alert type="info" class="mb-3">{{ $t("rconManager.templateTip") }}</n-alert>
    <n-flex justify="space-between">
      <n-upload v-if="!editingCommandUUID" action="/api/rcon/import" name="file" accept=".txt"
        :headers="uploadHeaders" :show-file-list="false" @finish="importFinished" @error="importFailed">
        <n-button secondary>{{ $t("button.import") }}</n-button>
      </n-upload>
      <span v-else />
      <n-flex>
        <n-button @click="commandModal = false">{{ $t("button.cancel") }}</n-button>
        <n-button type="primary" @click="saveCommand">{{ $t("button.save") }}</n-button>
      </n-flex>
    </n-flex>
  </n-modal>

  <!-- task modal -->
  <n-modal v-model:show="taskModal" preset="card"
    :title="editingTaskUUID ? $t('rconManager.editTask') : $t('rconManager.addTask')"
    style="width:min(92vw,620px)">
    <n-form label-placement="top">
      <n-form-item :label="$t('rconManager.taskName')" required>
        <n-input v-model:value="taskForm.name" />
      </n-form-item>
      <n-form-item label="命令来源">
        <n-radio-group v-model:value="taskCmdMode" size="small">
          <n-radio-button value="custom">自定义命令库</n-radio-button>
          <n-radio-button value="inline">直接输入命令</n-radio-button>
        </n-radio-group>
      </n-form-item>
      <n-form-item v-if="taskCmdMode === 'custom'" :label="$t('rconManager.boundCommand')" required>
        <n-select v-model:value="taskForm.rcon_uuid" :options="commandOptions" @update:value="selectTaskCommand"
          :placeholder="commands.length ? '选择已保存的自定义命令' : '暂无自定义命令，请切换为直接输入'" />
      </n-form-item>
      <n-form-item v-else label="RCON 命令" required>
        <n-input v-model:value="taskForm.inlineCmd" placeholder="例如: Save  或  Shutdown 60 AutoRestart"
          style="font-family:monospace" />
        <template #feedback>
          <n-text depth="3" style="font-size:11px">直接填写 RCON 命令，将自动存入命令库。可点击左侧指令台的命令名称参考格式。</n-text>
        </template>
      </n-form-item>
      <n-form-item :label="$t('rconManager.argumentContent')">
        <n-input v-model:value="taskForm.content" type="textarea" autosize :placeholder="$t('rconManager.taskContentTip')" />
      </n-form-item>
      <n-form-item :label="$t('rconManager.scheduleLabel')" required>
        <n-select v-model:value="scheduleType" :options="scheduleTypeOptions" />
      </n-form-item>
      <n-form-item v-if="scheduleType === 'interval'" :label="$t('rconManager.intervalMinutes')">
        <n-input-number v-model:value="intervalMinutes" :min="1" :max="1440" class="w-full" />
      </n-form-item>
      <n-form-item v-else-if="scheduleType === 'daily'" :label="$t('rconManager.dailyTime')">
        <n-input v-model:value="dailyTime" type="time" />
      </n-form-item>
      <template v-else-if="scheduleType === 'weekly'">
        <n-form-item :label="$t('rconManager.weekdayLabel')">
          <n-select v-model:value="weeklyDay" :options="weekOptions" />
        </n-form-item>
        <n-form-item :label="$t('rconManager.dailyTime')">
          <n-input v-model:value="weeklyTime" type="time" />
        </n-form-item>
      </template>
      <n-form-item v-else :label="$t('rconManager.cronExpression')">
        <n-input v-model:value="customCron" placeholder="0 4 * * *" />
      </n-form-item>
      <n-form-item :label="$t('rconManager.enabled')">
        <n-switch v-model:value="taskForm.enabled" />
      </n-form-item>
    </n-form>
    <n-flex justify="end">
      <n-button @click="taskModal = false">{{ $t("button.cancel") }}</n-button>
      <n-button type="primary" @click="saveTask">{{ $t("button.save") }}</n-button>
    </n-flex>
  </n-modal>
</template>

<style scoped lang="less">
.console-layout {
  display: grid;
  grid-template-columns: 210px 1fr;
  gap: 10px;
  height: calc(100vh - 130px);
  min-height: 400px;
}
.cmd-panel {
  display: flex;
  flex-direction: column;
  gap: 6px;
  border: 1.5px solid var(--n-border-color);
  border-radius: 8px;
  padding: 8px;
  overflow: hidden;
}
.cmd-search-wrap { position: relative; flex-shrink: 0; }
.cmd-search-icon {
  position: absolute; left: 9px; top: 50%;
  transform: translateY(-50%); opacity: 0.4; pointer-events: none;
}
.cmd-search-input {
  width: 100%; padding: 6px 8px 6px 28px;
  border: 1.5px solid var(--n-border-color); border-radius: 8px;
  background: transparent; color: inherit; font-size: 12px;
  outline: none; box-sizing: border-box;
  &:focus { border-color: var(--n-primary-color); }
}
.cmd-list { flex: 1; overflow-y: auto; display: flex; flex-direction: column; }
.cmd-group-label {
  font-size: 10px; font-weight: 800; opacity: 0.45;
  padding: 8px 6px 2px; letter-spacing: 0.3px; text-transform: uppercase;
}
.cmd-item {
  border: none; background: transparent; text-align: left;
  cursor: pointer; padding: 5px 6px; border-radius: 6px; color: inherit;
  transition: background 0.12s;
  &:hover { background: rgba(128,128,128,.1); }
}
.cmd-name {
  display: block; font-family: monospace; font-size: 12px;
  font-weight: 600; color: var(--n-primary-color);
}
.cmd-name--star { display: inline-flex; align-items: center; gap: 3px; }
.cmd-name--danger { color: #d03050; }
.cmd-danger-tag {
  font-family: sans-serif; font-size: 10px; font-weight: 700;
  background: rgba(208,48,80,.1); color: #d03050;
  border: 1px solid rgba(208,48,80,.25); border-radius: 3px;
  padding: 0 4px; margin-left: 3px; vertical-align: middle;
}
.cmd-desc { display: block; font-size: 11px; opacity: 0.5; line-height: 1.3; margin-top: 1px; }
.cmd-empty { text-align: center; opacity: 0.35; font-size: 12px; padding: 20px 0; }

.terminal-panel { display: flex; flex-direction: column; gap: 8px; overflow: hidden; }
.terminal-log {
  flex: 1; border: 1.5px solid var(--n-border-color); border-radius: 8px;
  padding: 10px 12px; font-family: monospace; font-size: 12.5px; min-height: 0;
}
.terminal-placeholder {
  opacity: 0.35; font-size: 12px; line-height: 1.7;
  text-align: center; padding: 40px 12px;
}
.terminal-entry {
  display: flex; gap: 6px; line-height: 1.6;
  padding: 1px 0; word-break: break-all;
}
.terminal-entry--cmd { opacity: 0.85; }
.terminal-entry--ok  { color: #18a058; }
.terminal-entry--err { color: #d03050; }
.terminal-time { font-size: 10px; opacity: 0.35; white-space: nowrap; flex-shrink: 0; padding-top: 2px; }
.terminal-prompt { flex-shrink: 0; font-weight: 700; opacity: 0.55; }
.terminal-prompt--ok  { color: #18a058; opacity: 1; }
.terminal-prompt--err { color: #d03050; opacity: 1; }
.terminal-text { flex: 1; white-space: pre-wrap; }
.terminal-input-bar {
  display: flex; align-items: center; gap: 8px;
  border: 2px solid var(--n-border-color); border-radius: 8px;
  padding: 6px 10px; flex-shrink: 0; transition: border-color 0.15s;
  &:focus-within { border-color: var(--n-primary-color); }
}
.terminal-input-icon { opacity: 0.35; flex-shrink: 0; }
.terminal-input {
  flex: 1; border: none; background: transparent; color: inherit;
  font-family: monospace; font-size: 13px; outline: none; min-width: 0;
}
.terminal-send-btn { flex-shrink: 0; }

.cmd-hint-bar {
  font-size: 11px; opacity: 0.55; padding: 3px 6px;
  border-left: 2px solid var(--n-primary-color);
  margin-top: 2px; font-family: monospace;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.terminal-obj { flex: 1; font-size: 12px; line-height: 1.5; }
.terminal-obj-row { display: flex; gap: 8px; padding: 1px 0; border-bottom: 1px solid rgba(128,128,128,.06); }
.terminal-obj-key { color: var(--n-primary-color); font-weight: 700; min-width: 140px; flex-shrink: 0; }
.terminal-obj-val { flex: 1; opacity: .9; display: flex; flex-wrap: wrap; gap: 6px; }
.terminal-obj-sub { background: rgba(128,128,128,.08); border-radius: 4px; padding: 0 6px; font-size: 11px; }
.terminal-obj-subkey { opacity: .6; }

/* autocomplete */
.terminal-input-wrap { position: relative; flex-shrink: 0; }
.ac-dropdown {
  position: absolute; bottom: calc(100% + 4px); left: 0; right: 0;
  background: var(--n-card-color, #1e1e2e); border: 1.5px solid var(--n-border-color);
  border-radius: 8px; z-index: 999; max-height: 220px; overflow-y: auto;
  box-shadow: 0 4px 16px rgba(0,0,0,.3); font-family: monospace; font-size: 12px;
}
.ac-item {
  padding: 5px 12px; cursor: pointer; line-height: 1.5;
  border-bottom: 1px solid rgba(128,128,128,.08);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  &:last-child { border-bottom: none; }
  &:hover { background: rgba(64,152,252,.12); }
}
.ac-item--active { background: rgba(64,152,252,.22); color: var(--n-primary-color); font-weight: 600; }

/* item list table */
.terminal-item-list {
  flex: 1; font-size: 12px; line-height: 1.4;
  border: 1px solid rgba(128,128,128,.15); border-radius: 6px; overflow: hidden;
  margin-top: 2px;
}
.til-header {
  display: grid; grid-template-columns: 1fr 1fr;
  background: rgba(64,152,252,.12); font-weight: 700; font-size: 11px;
  padding: 4px 8px; border-bottom: 1px solid rgba(128,128,128,.15);
}
.til-row {
  display: grid; grid-template-columns: 1fr 1fr;
  padding: 3px 8px; border-bottom: 1px solid rgba(128,128,128,.08);
  transition: background .1s;
  &:last-child { border-bottom: none; }
  &:hover { background: rgba(64,152,252,.08); }
}
.til-col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.til-en { font-family: monospace; color: var(--n-primary-color); opacity: .85; }
.til-zh { opacity: .8; }


@media (max-width: 560px) {
  .console-layout {
    grid-template-columns: 1fr;
    grid-template-rows: 170px 1fr;
  }
}
</style>
