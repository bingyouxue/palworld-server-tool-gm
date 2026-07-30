<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import dayjs from "dayjs";
import { useI18n } from "vue-i18n";
import { useMessage } from "naive-ui";
import ApiService from "@/service/api";
import PalDefenderConfig from "@/components/PalDefenderConfig.vue";

const props = defineProps({
  serverInfo: { type: Object, default: () => ({}) },
  serverMetrics: { type: Object, default: () => ({}) },
  metricsHistory: { type: Array, default: () => [] },
  players: { type: Array, default: () => [] },
});
const emit = defineEmits([
  "open-rcon",
  "open-backup",
  "open-broadcast",
  "open-config",
  "open-game-config",
  "refresh-server",
]);
const { t } = useI18n();
const api = new ApiService();
const message = useMessage();
const loading = ref(false);
const onlinePlayers = ref([]);
const backups = ref([]);
const tasks = ref([]);
const asArray = (value) => (Array.isArray(value) ? value : []);

const latestBackup = computed(() =>
  [...backups.value].sort((a, b) => new Date(b.save_time) - new Date(a.save_time))[0],
);
const activeTasks = computed(() => tasks.value.filter((task) => task.enabled));
const nextTask = computed(() =>
    activeTasks.value
      .filter((task) => task.next_run_at)
      .sort((a, b) => new Date(a.next_run_at) - new Date(b.next_run_at))[0],
);
const uptime = computed(() => {
  const seconds = Number(props.serverMetrics?.uptime || 0);
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  return days > 0
    ? t("overview.uptimeDays", { days, hours })
    : t("overview.uptimeHours", { hours });
});
const healthType = computed(() => {
  if (props.serverInfo?.running !== true && !props.serverInfo?.name) return "warning";
  if (props.serverInfo?.management_available === false) return "warning";
  const fps = Number(props.serverMetrics?.server_fps || 0);
  return fps > 0 && fps < 30 ? "warning" : "success";
});

const loadOverview = async () => {
  loading.value = true;
  try {
    const [onlineResponse, backupResponse, taskResponse] = await Promise.all([
      api.getOnlinePlayerList(),
      api.getBackupList({}),
      api.getRconTasks(),
    ]);
    onlinePlayers.value = asArray(onlineResponse.data.value);
    backups.value = asArray(backupResponse.data.value);
    tasks.value = asArray(taskResponse.data.value);
  } finally {
    loading.value = false;
  }
};

const formatTime = (value) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "—";

// ── 服务器管理弹窗 ──────────────────────────────────────
const showServerMgmt = ref(false);
const showPdConfig = ref(false);
// Always reflect the parent prop; can be overridden locally while polling
const isServerRunning = (info) => info?.running === true || !!info?.name;
const localServerRunning = ref(isServerRunning(props.serverInfo));
const serverRunning = computed(() => localServerRunning.value);
const serverPlatform = computed(() => props.serverInfo?.platform || "unknown");
// Watch prop so opening the modal always shows the correct current state
watch(() => props.serverInfo, (val) => {
  // Only sync if we're not in the middle of a poll (poll sets it independently)
  if (!statusPollTimer) {
    localServerRunning.value = isServerRunning(val);
  }
}, { deep: true, immediate: true });

const mgmtLoading = ref({ stop: false, start: false, restart: false, paldefender: false, ue4ss: false, serverUpdate: false });
const serverVersion = ref({
  gameVersion: '',
  installedBuildId: '',
  latestBuildId: '',
  hasUpdate: false,
  loading: false,
  error: '',
});
const displayedGameVersion = computed(() => serverVersion.value.gameVersion || props.serverInfo?.version || '无法读取（启动服务器后可获取）');

const checkServerVersion = async () => {
  if (serverVersion.value.loading) return;
  serverVersion.value.loading = true;
  serverVersion.value.error = '';
  try {
    const res = await fetch('/api/setup/server-version', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('palworld_token') || ''}` },
      body: JSON.stringify({}),
    });
    const json = await res.json();
    serverVersion.value.gameVersion = json.game_version || props.serverInfo?.version || '';
    serverVersion.value.installedBuildId = json.installed_build_id || '';
    serverVersion.value.latestBuildId = json.latest_build_id || '';
    serverVersion.value.hasUpdate = !!json.has_update;
    if (!res.ok) serverVersion.value.error = json.error || '版本检测失败';
  } catch (e) {
    serverVersion.value.error = e.message;
  } finally {
    serverVersion.value.loading = false;
  }
};

watch(showServerMgmt, (visible) => {
  if (visible) checkServerVersion();
});

// Plugin install status (checked via /api/server call — if server responds it's running,
// and we detect paldefender via a test RCON command that only works when PD is installed)
const pluginStatus = ref({ paldefender: null, ue4ss: null, paldefender_version: null, ue4ss_version: null }); // null=未检测
const pluginChecking = ref(false);

// DLL version info loaded from /api/paldefender/version
const pdVersion = ref({ dll_version: '', marker_version: '', latest_version: '', has_update: false, loading: false, error: '' });

const checkPdVersion = async () => {
  pdVersion.value.loading = true;
  pdVersion.value.error = '';
  try {
    const res = await fetch('/api/paldefender/version');
    const j = await res.json();
    if (res.ok) {
      pdVersion.value.dll_version    = j.dll_version    || '';
      pdVersion.value.marker_version = j.marker_version || '';
      pdVersion.value.latest_version = j.latest_version || '';
      pdVersion.value.has_update     = !!j.has_update;
      // If the DLL version endpoint found an actual version, the plugin is installed
      if (j.dll_version && j.dll_version !== 'unknown') {
        pluginStatus.value.paldefender = true;
        if (!pluginStatus.value.paldefender_version) {
          pluginStatus.value.paldefender_version = j.marker_version || j.dll_version;
        }
      }
    } else {
      pdVersion.value.error = j.error || '查询失败';
    }
  } catch (e) {
    pdVersion.value.error = e.message;
  } finally {
    pdVersion.value.loading = false;
  }
};

const checkPluginStatus = async () => {
  pluginChecking.value = true;
  try {
    const { data } = await api.getServerPlugins();
    if (data.value) {
      pluginStatus.value.paldefender = !!data.value.paldefender;
      pluginStatus.value.ue4ss = !!data.value.ue4ss;
      pluginStatus.value.paldefender_version = data.value.paldefender_version || null;
      pluginStatus.value.ue4ss_version = data.value.ue4ss_version || null;
    }
  } catch {
    pluginStatus.value.paldefender = false;
    pluginStatus.value.ue4ss = false;
  } finally {
    pluginChecking.value = false;
  }
  // Always refresh DLL version after plugin status check
  checkPdVersion();
  // If pluginStatus is still null after the API call (e.g. server offline),
  // resolve it so the badge does not stay stuck on "检测中" forever.
  if (pluginStatus.value.paldefender === null) pluginStatus.value.paldefender = false;
  if (pluginStatus.value.ue4ss === null) pluginStatus.value.ue4ss = false;
};

// Poll server status after shutdown/restart until it changes
let statusPollTimer = null;
const startStatusPoll = (expectRunning) => {
  if (statusPollTimer) clearInterval(statusPollTimer);
  let attempts = 0;
  statusPollTimer = setInterval(async () => {
    attempts++;
    try {
      const { data } = await api.getServerInfo();
      const nowRunning = isServerRunning(data.value);
      localServerRunning.value = nowRunning;
      if (nowRunning === expectRunning || attempts >= 20) {
        clearInterval(statusPollTimer);
        statusPollTimer = null;
      }
    } catch {
      localServerRunning.value = false;
      if (!expectRunning) {
        clearInterval(statusPollTimer);
        statusPollTimer = null;
      }
    }
  }, 3000);
};

const handleStopServer = async () => {
  mgmtLoading.value.stop = true;
  try {
    const cmd = { seconds: 10, message: "Server is shutting down" };
    console.log("[ServerMgmt] shutdown →", cmd);
    const { data, statusCode } = await api.shutdownServer(cmd);
    console.log("[ServerMgmt] shutdown response:", statusCode.value, data.value);
    if (statusCode.value === 200) {
      if (data.value?.forced) {
        message.warning("REST API 不可用，已通过本机进程树强制停止服务器");
        localServerRunning.value = false;
        startStatusPoll(false);
        setTimeout(() => emit("refresh-server"), 1000);
      } else {
        message.success("关闭命令已发送，服务器将在 10 秒后停止");
        setTimeout(() => {
          startStatusPoll(false);
          setTimeout(() => emit("refresh-server"), 15000);
        }, 12000);
      }
    } else {
      message.error(data.value?.error || "停止失败");
    }
  } catch (e) {
    console.error("[ServerMgmt] shutdown error:", e);
    message.error("停止失败: " + e.message);
  } finally {
    mgmtLoading.value.stop = false;
  }
};

const handleStartServer = async (mode = "silent") => {
  mgmtLoading.value.start = true;
  try {
    console.log("[ServerMgmt] start → POST /api/server/start mode:", mode);
    const { data, statusCode } = await api.startServer(mode);
    console.log("[ServerMgmt] start response:", statusCode.value, data.value);
    if (statusCode.value === 200) {
      const modeLabel = mode === "cmd" ? "Cmd 模式" : "静默模式";
      const logHint = data.value?.log_path ? `\n服务端日志：${data.value.log_path}` : "";
      message.success(`启动命令已发送（${modeLabel}），等待服务器上线...${logHint}`, { duration: 8000 });
      startStatusPoll(true);
      setTimeout(() => emit("refresh-server"), 20000);
    } else {
      const err = data.value?.error || "启动失败";
      message.error(err);
      console.warn("[ServerMgmt] start failed:", err);
    }
  } catch (e) {
    console.error("[ServerMgmt] start error:", e);
    message.error("启动失败: " + e.message);
  } finally {
    mgmtLoading.value.start = false;
  }
};

const handleRestartServer = async () => {
  mgmtLoading.value.restart = true;
  try {
    const cmd = { seconds: 10, message: "Server is restarting" };
    console.log("[ServerMgmt] restart →", cmd);
    const { data, statusCode } = await api.restartServer(cmd);
    console.log("[ServerMgmt] restart response:", statusCode.value, data.value);
    if (statusCode.value === 200) {
      const modeLabel = data.value?.mode === "cmd" ? "Cmd 模式" : "静默模式";
      if (data.value?.graceful === false) {
        message.warning(`RCON 不可用，将强制停止后以${modeLabel}重新启动`, { duration: 8000 });
      } else {
        message.success(`重启命令已发送，服务器将在 10 秒后停止并以${modeLabel}重新启动`, { duration: 8000 });
      }
      setTimeout(() => {
        startStatusPoll(true);
        setTimeout(() => emit("refresh-server"), 30000);
      }, 12000);
    } else {
      message.error(data.value?.error || "重启失败");
    }
  } catch (e) {
    console.error("[ServerMgmt] restart error:", e);
    message.error("重启失败: " + e.message);
  } finally {
    mgmtLoading.value.restart = false;
  }
};

const handleInstallPalDefender = async () => {
  await doInstallMod('paldefender');
};

const handleInstallUE4SS = async () => {
  await doInstallMod('ue4ss');
};

const modProgress = ref({ show: false, title: '', lines: [], done: false, failed: false });

const doInstallMod = async (component) => {
  const key = component === 'paldefender' ? 'paldefender' : 'ue4ss';
  mgmtLoading.value[key] = true;
  modProgress.value = { show: true, title: component === 'paldefender' ? 'PalDefender' : 'UE4SS', lines: [], done: false, failed: false };
  try {
    const res = await fetch('/api/server/mods/install', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('palworld_token') || ''}` },
      body: JSON.stringify({ component, channel: 'stable' }),
    });
    const json = await res.json();
    if (!res.ok) { message.error(json.error || '启动安装失败'); return; }
    const installId = json.install_id;
    const token = localStorage.getItem('palworld_token') || '';
    const es = new EventSource(`/api/server/mods/install/progress/${installId}?token=${encodeURIComponent(token)}`);
    es.addEventListener('log', (e) => {
      const line = e.data || '';
      modProgress.value.lines.push(line);
      if (line.startsWith('[错误]')) modProgress.value.failed = true;
    });
    es.addEventListener('done', () => { modProgress.value.done = true; es.close(); checkPluginStatus(); checkPdVersion(); mgmtLoading.value[key] = false; });
    es.addEventListener('error', (e) => { modProgress.value.lines.push('[错误] ' + (e.data || '连接断开')); modProgress.value.failed = true; modProgress.value.done = true; es.close(); mgmtLoading.value[key] = false; });
  } catch (e) {
    message.error('请求失败: ' + e.message);
    mgmtLoading.value[key] = false;
  }
};

const doServerUpdate = async () => {
  mgmtLoading.value.serverUpdate = true;
  modProgress.value = { show: true, title: '幻兽帕鲁服务端更新', lines: [], done: false, failed: false };
  try {
    const res = await fetch('/api/setup/server-update', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${localStorage.getItem('palworld_token') || ''}` },
      body: JSON.stringify({}),
    });
    const json = await res.json();
    if (!res.ok) { message.error(json.error || '启动更新失败'); modProgress.value.done = true; return; }
    const installId = json.install_id;
    const es = new EventSource(`/api/setup/install/progress/${installId}`);
    es.addEventListener('log', (e) => {
      const line = e.data || '';
      modProgress.value.lines.push(line);
      if (line.startsWith('[错误]')) modProgress.value.failed = true;
    });
    es.addEventListener('done', () => {
      modProgress.value.done = true;
      es.close();
      mgmtLoading.value.serverUpdate = false;
      if (modProgress.value.failed) message.error('服务端更新失败，请查看更新日志');
      else {
        message.success('服务端已更新并通过最新 Build ID 校验');
        checkServerVersion();
      }
    });
    es.addEventListener('error', (e) => { modProgress.value.lines.push('[错误] ' + (e.data || '连接断开')); modProgress.value.failed = true; modProgress.value.done = true; es.close(); mgmtLoading.value.serverUpdate = false; });
  } catch (e) {
    message.error('请求失败: ' + e.message);
    modProgress.value.done = true;
    mgmtLoading.value.serverUpdate = false;
  }
};

onMounted(() => {
  loadOverview();
  checkPluginStatus();
  checkPdVersion();
});
onUnmounted(() => {
  if (statusPollTimer) clearInterval(statusPollTimer);
});

// ── 效能分析弹窗 ────────────────────────────────────────────────────────────
const showPerfModal = ref(false);
const showCoreModal = ref(false);
const selectedCore = ref(0);

const fmtUptime = (sec) => {
  const s = Math.max(0, Math.floor(Number(sec) || 0));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d} 天 ${h} 时`;
  if (h > 0) return `${h} 时 ${m} 分`;
  if (m > 0) return `${m} 分`;
  return `${s} 秒`;
};

// SVG 走势图：把数值数组渲染成 polyline 坐标串
const buildTrendPath = (values, W, H, fixedMax = null) => {
  const known = values.filter((v) => v != null && Number.isFinite(v));
  if (known.length < 2) return { line: "", area: "", segments: [] };
  const max = fixedMax ?? Math.max(...known, 1);
  const pts = values.map((v, i) => {
    if (v == null || !Number.isFinite(v)) return null;
    const x = ((i / (values.length - 1)) * W).toFixed(1);
    const y = (H - Math.max(0, Math.min(v / max, 1)) * (H - 4) - 2).toFixed(1);
    return `${x},${y}`;
  });
  const segments = [];
  let seg = [];
  for (const p of pts) {
    if (p == null) { if (seg.length) segments.push(seg); seg = []; }
    else seg.push(p);
  }
  if (seg.length) segments.push(seg);
  const validPts = pts.filter(Boolean);
  const area = validPts.length >= 2
    ? `0,${H} ${validPts.join(" ")} ${W},${H}`
    : "";
  return { segments, area };
};

const fpsTrend = computed(() => {
  const vals = props.metricsHistory.map((h) => h.fps != null ? Number(h.fps) : null);
  return buildTrendPath(vals, 260, 60, Math.max(60, ...vals.filter((v) => v != null)));
});
const cpuTrend = computed(() => {
  const vals = props.metricsHistory.map((h) => h.cpu != null ? Number(h.cpu) : null);
  const known = vals.filter((v) => v != null && Number.isFinite(v));
  return buildTrendPath(vals, 260, 60, Math.max(5, ...known.map((v) => v * 1.25)));
});
const memTrend = computed(() => {
  const vals = props.metricsHistory.map((h) => h.memPct != null ? Number(h.memPct) * 100 : null);
  const known = vals.filter((v) => v != null && Number.isFinite(v));
  return buildTrendPath(vals, 260, 60, Math.max(5, ...known.map((v) => v * 1.25)));
});

const latestPerCore = computed(() => Array.isArray(props.serverMetrics?.cpu_per_core) ? props.serverMetrics.cpu_per_core : []);
const averageCoreLoad = computed(() => latestPerCore.value.length
  ? latestPerCore.value.reduce((sum, value) => sum + Number(value || 0), 0) / latestPerCore.value.length
  : 0);
const coreLoadColor = (value) => {
  const load = Number(value) || 0;
  if (load >= 80) return "#e85d75";
  if (load >= 50) return "#f0a020";
  if (load >= 20) return "#36ad6a";
  return "#4098fc";
};
const coreTrend = computed(() => buildTrendPath(
  props.metricsHistory.map((h) => Array.isArray(h.perCore) && h.perCore[selectedCore.value] != null ? Number(h.perCore[selectedCore.value]) : null),
  260, 80, 100,
));

const fmtBytes = (n) => {
  if (!n) return "0 B";
  if (n >= 1 << 30) return (n / (1 << 30)).toFixed(1) + " GB";
  if (n >= 1 << 20) return Math.round(n / (1 << 20)) + " MB";
  return Math.round(n / 1024) + " KB";
};
</script>
<template>
  <n-scrollbar class="h-full">
    <div class="p-5 max-w-1400px mx-auto">
      <n-flex justify="space-between" align="center" class="mb-4">
        <div>
          <n-h2 class="m-0">{{ $t("overview.title") }}</n-h2>
          <n-text depth="3">{{ $t("overview.subtitle") }}</n-text>
        </div>
        <n-button secondary :loading="loading" @click="loadOverview(); emit('refresh-server')">
          {{ $t("overview.refresh") }}
        </n-button>
      </n-flex>

      <n-grid cols="1 640:2 1050:4" :x-gap="16" :y-gap="16">
        <n-gi><n-card size="small">
            <n-statistic :label="$t('overview.serverStatus')">
              <n-flex align="center">
                <n-badge dot :type="healthType" />
              <n-text strong>{{ serverInfo?.name || (serverInfo?.running ? "服务端已运行（管理接口不可用）" : $t("status.serverUnavailable")) }}</n-text>
              </n-flex>
            </n-statistic>
            <n-text depth="3">{{ serverInfo?.version || "—" }}</n-text>
        </n-card></n-gi>
        <n-gi><n-card size="small">
          <n-statistic :label="$t('overview.onlinePlayers')" :value="serverMetrics?.current_player_num ?? onlinePlayers.length">
            <template #suffix>/ {{ serverMetrics?.max_player_num ?? "—" }}</template>
            </n-statistic>
          <n-text depth="3">{{ $t("overview.totalPlayers", { count: players.length }) }}</n-text>
        </n-card></n-gi>
        <n-gi><n-card size="small" style="cursor:pointer" @click="showPerfModal = true">
          <n-statistic :label="$t('item.serverFps')" :value="serverMetrics?.server_fps ?? '—'" />
          <n-text depth="3">{{ $t("item.serverFrameTime") }}: {{ serverMetrics?.server_frame_time ?? "—" }} ms</n-text>
        </n-card></n-gi>
        <n-gi><n-card size="small">
            <n-statistic :label="$t('item.serverUptime')" :value="uptime" />
          <n-text depth="3">{{ $t("item.serverDays") }}: {{ serverMetrics?.days ?? "—" }}</n-text>
        </n-card></n-gi>
      </n-grid>

      <n-grid cols="1 760:2" :x-gap="16" :y-gap="16" class="mt-4">
        <n-gi>
          <n-card class="overview-panel" :title="$t('overview.operations')">
            <n-grid cols="2 560:4" :x-gap="12" :y-gap="12">
              <n-gi><n-button block type="primary" secondary @click="emit('open-rcon')">{{ $t("button.rcon") }}</n-button></n-gi>
              <n-gi><n-button block type="success" secondary @click="emit('open-backup')">{{ $t("button.backup") }}</n-button></n-gi>
              <n-gi><n-button block type="warning" secondary @click="emit('open-broadcast')">{{ $t("button.broadcast") }}</n-button></n-gi>
              <n-gi><n-button block secondary @click="emit('open-config')">{{ $t("configuration.title") }}</n-button></n-gi>
              <n-gi><n-button block secondary type="info" @click="emit('open-game-config')">{{ $t("button.gameConfig") }}</n-button></n-gi>
              <n-gi>
                <n-button block type="error" secondary @click="showServerMgmt = true">{{ $t("button.serverMgmt") }}</n-button>
              </n-gi>
            </n-grid>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card class="overview-panel" :title="$t('overview.automation')">
            <n-descriptions :column="1" label-placement="left">
              <n-descriptions-item :label="$t('overview.activeTasks')">{{ activeTasks.length }}</n-descriptions-item>
              <n-descriptions-item :label="$t('overview.nextTask')">{{ nextTask?.name || "—" }}</n-descriptions-item>
              <n-descriptions-item :label="$t('overview.nextRun')">{{ formatTime(nextTask?.next_run_at) }}</n-descriptions-item>
            </n-descriptions>
          </n-card>
        </n-gi>
      </n-grid>

      <n-grid cols="1 760:2" :x-gap="16" :y-gap="16" class="mt-4">
        <n-gi>
          <n-card class="overview-panel overview-panel--online" :title="$t('overview.onlineNow')">
            <n-empty v-if="onlinePlayers.length === 0" :description="$t('overview.noOnlinePlayers')" />
            <n-list v-else hoverable>
              <n-list-item v-for="player in onlinePlayers.slice(0, 6)" :key="player.player_uid">
                <n-flex justify="space-between">
                  <n-text>{{ player.nickname }}</n-text>
                  <n-tag size="small" type="success">Lv.{{ player.level }}</n-tag>
                </n-flex>
              </n-list-item>
            </n-list>
          </n-card>
        </n-gi>
        <n-gi>
          <n-card class="overview-panel" :title="$t('overview.backupStatus')">
            <n-empty v-if="!latestBackup" :description="$t('overview.noBackup')" />
            <n-descriptions v-else :column="1" label-placement="left">
              <n-descriptions-item :label="$t('overview.latestBackup')">{{ formatTime(latestBackup.save_time) }}</n-descriptions-item>
              <n-descriptions-item :label="$t('overview.backupCount')">{{ backups.length }}</n-descriptions-item>
            </n-descriptions>
          </n-card>
        </n-gi>
      </n-grid>
      <!-- Credits 区块 -->
      <div class="credits-block">
        <div class="credits-inner">
          <img src="/logo.jpg" alt="logo" class="credits-logo" />
          <div class="credits-text">
            <div class="credits-made">
              <span class="credits-label">{{ $t("credits.madeBy") }}</span>
              <n-tooltip trigger="hover" placement="top" :delay="300">
                <template #trigger>
                  <span class="credits-author">青山原不老</span>
                </template>
                QQ：81342543
              </n-tooltip>
            </div>
            <div class="credits-row">
              <span class="credits-label">{{ $t("credits.poweredBy") }}</span>
              <a href="https://github.com/zaigie/palworld-server-tool" target="_blank" rel="noopener" class="credits-link">PST</a>
            </div>
            <div class="credits-row">
              <span class="credits-label">{{ $t("credits.thanks") }}</span>
              <a href="https://github.com/Ultimeit/PalDefender" target="_blank" rel="noopener" class="credits-link">PalDefender</a>
              <span class="credits-sep">·</span>
              <a href="https://github.com/io-software-ai/palserver-gui" target="_blank" rel="noopener" class="credits-link">palserver-gui</a>
            </div>
          </div>
        </div>
      </div>
    </div>
  </n-scrollbar>

  <!-- 服务器管理弹窗 -->
  <n-modal v-model:show="showServerMgmt" preset="card" :title="$t('serverMgmt.title')"
    style="width:90%;max-width:520px" :bordered="false">
    <div class="server-mgmt">
      <n-alert type="info" :show-icon="true" size="small" class="mb-3">
        {{ $t("serverMgmt.currentStatus") }}：
        <n-tag :type="serverRunning ? 'success' : 'error'" size="small" round style="margin-left:8px">
          {{ serverRunning ? $t("serverMgmt.running") : $t("serverMgmt.stopped") }}
        </n-tag>
      </n-alert>

      <n-card size="small" :title="$t('serverMgmt.serverControl')" class="mb-3">
        <n-alert v-if="serverRunning && serverInfo?.management_available === false" type="warning" :show-icon="true" size="small" class="mb-3">
          游戏服务端正在运行，但官方 Palworld REST API 不可用。PST 管理接口需要官方 REST（当前配置为 8212）；PalDefender 控制台显示的 REST 端口（如 17973）是插件 API，不能替代官方接口。请检查 PalWorldSettings.ini 中 RESTAPIEnabled=True、RESTAPIPort 与 PST 配置一致，并确认 AdminPassword 正确后重启服务端。
          <div v-if="serverInfo?.management_error" style="margin-top:4px;font-family:monospace;font-size:11px;word-break:break-all">
            {{ serverInfo.management_error }}
          </div>
        </n-alert>
        <n-space vertical>
          <n-flex align="center" justify="space-between">
            <div>
              <div class="mgmt-label">{{ serverRunning ? $t("button.stopServer") : $t("button.startServer") }}</div>
              <n-text depth="3" class="mgmt-desc">
                {{ serverRunning ? $t("serverMgmt.stopDesc") : $t("serverMgmt.startDesc") }}
              </n-text>
            </div>
            <n-button v-if="serverRunning" type="error" :loading="mgmtLoading.stop"
              @click="handleStopServer" size="small" round>{{ $t("button.stopServer") }}</n-button>
            <n-flex v-else :size="6">
              <n-button type="success" :loading="mgmtLoading.start"
                @click="handleStartServer('silent')" size="small" round>
                静默启动
              </n-button>
              <n-button type="info" :loading="mgmtLoading.start"
                @click="handleStartServer('cmd')" size="small" round>
                Cmd 启动
              </n-button>
            </n-flex>
          </n-flex>
          <n-divider style="margin:8px 0" />
          <n-flex align="center" justify="space-between">
            <div>
              <div class="mgmt-label">{{ $t("button.restartServer") }}</div>
              <n-text depth="3" class="mgmt-desc">{{ $t("serverMgmt.restartDesc") }}</n-text>
            </div>
            <n-button type="warning" :loading="mgmtLoading.restart"
              :disabled="!serverRunning" @click="handleRestartServer" size="small" round>
              {{ $t("button.restartServer") }}
            </n-button>
          </n-flex>
        </n-space>
      </n-card>

      <!-- 服务端更新卡片 -->
      <n-card size="small" class="mb-3">
        <template #header>
          <n-flex align="center" justify="space-between">
            <span>幻兽帕鲁服务端</span>
          </n-flex>
        </template>
        <n-space vertical :size="8">
          <div class="server-version-panel">
            <n-flex align="center" :size="8" wrap>
              <n-text strong>Game version is {{ displayedGameVersion }}</n-text>
              <n-tag v-if="!serverVersion.loading && !serverVersion.error && !serverVersion.hasUpdate && serverVersion.latestBuildId" type="success" size="small" round>
                已是最新版本
              </n-tag>
              <n-tag v-else-if="serverVersion.hasUpdate" type="warning" size="small" round>
                检测到新版本
              </n-tag>
              <n-spin v-if="serverVersion.loading" size="small" />
            </n-flex>
            <n-text depth="3" class="build-id-line">
              当前 Build ID：{{ serverVersion.installedBuildId || '未找到本地 appmanifest' }}
            </n-text>
            <n-alert v-if="serverVersion.hasUpdate && serverVersion.latestBuildId" type="warning" :show-icon="true" size="small" style="margin-top:8px">
              Steam public 分支最新 Build ID：<strong>{{ serverVersion.latestBuildId }}</strong>，当前服务端需要更新。
            </n-alert>
            <n-text v-else-if="serverVersion.error" type="error" class="version-error">
              最新版本检测失败：{{ serverVersion.error }}
            </n-text>
          </div>
          <n-text depth="3" style="font-size:12px">通过 SteamCMD 自动下载最新版本并更新到服务器目录（等同于 app_update 2394010 validate）。更新前请先停止服务器。</n-text>
          <n-flex :size="6">
            <n-button size="tiny" :type="serverVersion.hasUpdate ? 'error' : 'warning'"
              :class="{ 'update-button-highlight': serverVersion.hasUpdate }"
              :disabled="serverRunning || serverVersion.loading" :title="serverRunning ? $t('serverMgmt.stopServerFirst') : ''"
              :loading="mgmtLoading.serverUpdate" @click="doServerUpdate">
              {{ serverVersion.hasUpdate ? '立即更新服务端' : '更新服务端' }}
            </n-button>
            <n-button size="tiny" secondary :loading="serverVersion.loading" @click="checkServerVersion">
              重新检测版本
            </n-button>
          </n-flex>
        </n-space>
      </n-card>

      <!-- 插件安装状态卡片 -->
      <n-card size="small" class="mb-3">
        <template #header>
          <n-flex align="center" justify="space-between">
            <span>{{ $t("serverMgmt.pluginMgmt") }}</span>
            <n-button text size="tiny" :loading="pluginChecking" @click="checkPluginStatus">
              {{ $t("serverMgmt.redetect") }}
            </n-button>
          </n-flex>
        </template>
        <n-space vertical :size="10">
          <n-text depth="3" style="font-size:11px;color:#e88080">⚠ 安装和更新前请停止服务器</n-text>
          <!-- PalDefender -->
          <div class="plugin-card">
            <n-flex align="flex-start" :wrap="false" :size="10">
              <svg viewBox="0 0 512 512" class="plugin-icon" fill="currentColor" style="color:#4098fc">
                <path d="M256 16c25 24 100 72 150 72v96c0 96-75 240-150 312-75-72-150-216-150-312V88c50 0 125-48 150-72z"/>
              </svg>
              <div style="flex:1;min-width:0">
                <n-flex align="center" justify="space-between" :size="8">
                  <span class="plugin-name">PalDefender 反外挂</span>
                  <!-- Installed: DLL found by version check OR plugin-status API -->
                  <span v-if="pdVersion.dll_version && pdVersion.dll_version !== 'unknown' || pluginStatus.paldefender === true"
                    class="plugin-badge installed">
                    <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                    {{ $t("serverMgmt.installed") }}
                    <span v-if="pdVersion.dll_version && pdVersion.dll_version !== 'unknown'" class="plugin-version">DLL {{ pdVersion.dll_version }}</span>
                    <span v-else-if="pluginStatus.paldefender_version" class="plugin-version">{{ pluginStatus.paldefender_version }}</span>
                    <!-- update badge inline, only when installed + update available -->
                    <n-tag v-if="pdVersion.has_update" type="warning" size="tiny" style="margin-left:4px">有新版本 {{ pdVersion.latest_version }}</n-tag>
                  </span>
                  <!-- Not installed: both checks done and neither found DLL -->
                  <span v-else-if="!pdVersion.loading && pluginStatus.paldefender === false" class="plugin-badge not-installed">
                    {{ $t("serverMgmt.notInstalled") }}
                  </span>
                  <!-- Checking: still waiting for at least one check to complete -->
                  <span v-else class="plugin-badge checking">{{ $t("serverMgmt.detecting") }}</span>
                </n-flex>
                <p class="plugin-desc">{{ $t("serverMgmt.paldefenderDesc") }}</p>
                <n-flex :size="6" class="mt-2" wrap>
                  <n-button v-if="pluginStatus.paldefender === true" size="tiny" type="primary"
                    :disabled="serverRunning" :title="serverRunning ? $t('serverMgmt.stopServerFirst') : ''"
                    @click="handleInstallPalDefender" :loading="mgmtLoading.paldefender">
                    {{ $t("button.update") }}
                  </n-button>
                  <n-button v-else size="tiny" type="primary"
                    :disabled="serverRunning" :title="serverRunning ? $t('serverMgmt.stopServerFirst') : ''"
                    @click="handleInstallPalDefender" :loading="mgmtLoading.paldefender">
                    {{ $t("button.install") }}
                  </n-button>
                </n-flex>
                <p class="plugin-note">{{ $t("serverMgmt.paldefenderNote") }}</p>
                <n-button size="tiny" secondary @click="showServerMgmt=false;showPdConfig=true">
                  配置文件编辑
                </n-button>
              </div>
            </n-flex>
          </div>

          <n-divider style="margin:6px 0" />

          <!-- UE4SS -->
          <div class="plugin-card">
            <n-flex align="flex-start" :wrap="false" :size="10">
              <svg viewBox="0 0 512 512" class="plugin-icon" fill="currentColor" style="color:#4098fc">
                <path d="M103.432 17.844c-1.118.005-2.234.032-3.348.08-2.547.11-5.083.334-7.604.678-20.167 2.747-39.158 13.667-52.324 33.67-24.613 37.4 2.194 98.025 56.625 98.025.536 0 1.058-.012 1.583-.022v.704h60.565c-10.758 31.994-30.298 66.596-52.448 101.43-2.162 3.4-4.254 6.878-6.29 10.406l34.878 35.733-56.263 9.423c-32.728 85.966-27.42 182.074 48.277 182.074v-.002l9.31.066c23.83-.57 46.732-4.298 61.325-12.887 4.174-2.458 7.63-5.237 10.467-8.42h-32.446c-20.33 5.95-40.8-6.94-47.396-25.922-8.956-25.77 7.52-52.36 31.867-60.452 5.803-1.93 11.723-2.834 17.565-2.834v-.406h178.33c-.57-44.403 16.35-90.125 49.184-126 23.955-26.176 42.03-60.624 51.3-94.846l-41.225-24.932 38.272-6.906-43.37-25.807h-.005l.002-.002.002.002 52.127-8.85c-5.232-39.134-28.84-68.113-77.37-68.113C341.14 32.26 222.11 35.29 149.34 28.496c-14.888-6.763-30.547-10.723-45.908-10.652z"/>
              </svg>
              <div style="flex:1;min-width:0">
                <n-flex align="center" justify="space-between" :size="8">
                  <span class="plugin-name">UE4SS 模组加载器</span>
                  <span v-if="pluginStatus.ue4ss === true" class="plugin-badge installed">
                    <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
                    {{ $t("serverMgmt.installed") }}
                    <span v-if="pluginStatus.ue4ss_version" class="plugin-version">{{ pluginStatus.ue4ss_version }}</span>
                  </span>
                  <span v-else-if="pluginStatus.ue4ss === false" class="plugin-badge not-installed">
                    {{ $t("serverMgmt.notInstalled") }}
                  </span>
                  <span v-else class="plugin-badge checking">{{ $t("serverMgmt.detecting") }}</span>
                </n-flex>
                <p class="plugin-desc">{{ $t("serverMgmt.ue4ssDesc") }}</p>
                <n-flex :size="6" class="mt-2" wrap>
                  <n-button v-if="pluginStatus.ue4ss === true" size="tiny" type="primary"
                    :disabled="serverRunning" :title="serverRunning ? $t('serverMgmt.stopServerFirst') : ''"
                    @click="handleInstallUE4SS" :loading="mgmtLoading.ue4ss">
                    {{ $t("button.update") }}
                  </n-button>
                  <n-button v-else size="tiny" type="primary"
                    :disabled="serverRunning" :title="serverRunning ? $t('serverMgmt.stopServerFirst') : ''"
                    @click="handleInstallUE4SS" :loading="mgmtLoading.ue4ss">
                    {{ $t("button.install") }}
                  </n-button>
                </n-flex>
              </div>
            </n-flex>
          </div>
        </n-space>
      </n-card>
    </div>
    <template #footer>
      <div style="text-align:right">
        <n-button @click="showServerMgmt = false" size="small">{{ $t("serverMgmt.close") }}</n-button>
      </div>
    </template>
  </n-modal>

  <!-- 模组安装进度弹窗 -->
  <n-modal v-model:show="showPdConfig" preset="card" title="PalDefender 配置编辑器"
    style="width:min(96vw,780px)" :segmented="true">
    <pal-defender-config :show="showPdConfig"/>
    <template #footer>
      <n-button size="small" @click="showPdConfig=false">关闭</n-button>
    </template>
  </n-modal>

  <n-modal v-model:show="modProgress.show" preset="card" :title="modProgress.done ? modProgress.title : '正在处理 ' + modProgress.title"
    style="width:90%;max-width:500px" :bordered="false" :closable="modProgress.done"
    :mask-closable="modProgress.done">
    <div class="mod-progress-log" ref="modLogEl">
      <div v-for="(line, i) in modProgress.lines" :key="i" class="mod-log-line">{{ line }}</div>
      <div v-if="!modProgress.done" class="mod-log-spinner">
        <n-spin size="small" /> 处理中，请稍候...
      </div>
      <div v-else-if="modProgress.failed" class="mod-log-failed">操作失败，请根据上方错误信息检查 SteamCMD、网络和安装目录。</div>
      <div v-else class="mod-log-done">操作完成，可以关闭此窗口</div>
    </div>
    <template #footer>
      <div style="text-align:right">
        <n-button :disabled="!modProgress.done" @click="modProgress.show = false" size="small" type="primary">
          关闭
        </n-button>
      </div>
    </template>
  </n-modal>

  <!-- 效能分析弹窗 -->
  <n-modal v-model:show="showPerfModal" preset="card" title="效能分析"
    style="width:min(94vw,820px);max-height:92vh"
    content-style="max-height:calc(92vh - 72px);overflow-y:auto;padding-bottom:20px;display:flex;flex-direction:column"
    :mask-closable="true">

    <!-- 概要数字卡 -->
    <n-grid :cols="2" :x-gap="12" :y-gap="12" class="mb-4" style="order:1">
      <n-gi>
        <n-card size="small">
          <n-statistic label="CPU 占用" style="cursor:pointer" @click="showCoreModal = true"
            :value="serverMetrics?.cpu_percent != null ? serverMetrics.cpu_percent.toFixed(1) + '%' : '—'" />
          <n-text depth="3" style="font-size:11px">
            共 {{ serverMetrics?.cpu_cores ?? '—' }} 个逻辑核心 · 占总算力 {{ serverMetrics?.cpu_total_percent != null ? serverMetrics.cpu_total_percent.toFixed(1) + '%' : '—' }} · 点击查看
          </n-text>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <n-statistic label="内存使用"
            :value="serverMetrics?.memory_bytes ? fmtBytes(serverMetrics.memory_bytes) : '—'" />
          <n-text depth="3" style="font-size:11px">
            / {{ serverMetrics?.memory_total_bytes ? fmtBytes(serverMetrics.memory_total_bytes) : '—' }}
          </n-text>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <n-statistic label="服务器 FPS"
            :value="serverMetrics?.server_fps ?? '—'" />
          <n-text depth="3" style="font-size:11px">
            帧时间: {{ serverMetrics?.server_frame_time ?? '—' }} ms
          </n-text>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card size="small">
          <n-statistic label="总计运行时间"
            :value="fmtUptime(serverMetrics?.uptime)" />
          <n-text depth="3" style="font-size:11px">
            {{ serverMetrics?.days ?? '—' }} 游戏天
          </n-text>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 资源占用 -->
    <n-card size="small" class="mb-4" style="order:3">
      <n-flex align="center" :size="8" class="mb-3">
        <svg stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24"
          stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
          <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
        </svg>
        <span style="font-size:13px;font-weight:800">资源占用</span>
      </n-flex>

      <!-- CPU -->
      <div class="mb-3">
        <n-flex justify="space-between" class="mb-1">
          <n-text depth="3" style="font-size:12px">CPU（占总算力）</n-text>
          <n-text strong style="font-size:12px">
            {{ serverMetrics?.cpu_total_percent != null ? serverMetrics.cpu_total_percent.toFixed(1) + '%' : '—' }}
          </n-text>
        </n-flex>
        <n-progress
          :percentage="serverMetrics?.cpu_total_percent != null ? Math.min(100, +serverMetrics.cpu_total_percent.toFixed(1)) : 0"
          :color="(serverMetrics?.cpu_percent ?? 0) > 80 ? '#e88080' : '#4098fc'"
          :rail-color="'rgba(128,128,128,0.15)'"
          :height="8" :border-radius="4" :show-indicator="false"
        />
      </div>

      <!-- 内存 -->
      <div class="mb-3">
        <n-flex justify="space-between" class="mb-1">
          <n-text depth="3" style="font-size:12px">内存</n-text>
          <n-text strong style="font-size:12px">
            {{ serverMetrics?.memory_bytes && serverMetrics?.memory_total_bytes
              ? fmtBytes(serverMetrics.memory_bytes) + ' / ' + fmtBytes(serverMetrics.memory_total_bytes)
              : '—' }}
          </n-text>
        </n-flex>
        <n-progress
          :percentage="serverMetrics?.memory_bytes && serverMetrics?.memory_total_bytes
            ? Math.min(100, Math.round(serverMetrics.memory_bytes / serverMetrics.memory_total_bytes * 100))
            : 0"
          :color="'#18a058'"
          :rail-color="'rgba(128,128,128,0.15)'"
          :height="8" :border-radius="4" :show-indicator="false"
        />
      </div>

      <!-- FPS 条 -->
      <div>
        <n-flex justify="space-between" class="mb-1">
          <n-text depth="3" style="font-size:12px">服务器 FPS</n-text>
          <n-text strong style="font-size:12px">
            {{ serverMetrics?.server_fps != null ? serverMetrics.server_fps + ' fps' : '—' }}
          </n-text>
        </n-flex>
        <n-progress
          :percentage="serverMetrics?.server_fps
            ? Math.min(100, Math.round(serverMetrics.server_fps / 60 * 100))
            : 0"
          :color="(serverMetrics?.server_fps ?? 60) < 30 ? '#e88080' : '#18a058'"
          :rail-color="'rgba(128,128,128,0.15)'"
          :height="8" :border-radius="4" :show-indicator="false"
        />
      </div>
    </n-card>
  </n-modal>

  <n-modal v-model:show="showCoreModal" preset="card" title="逻辑核心实时占用"
    class="core-modal" style="width:min(94vw,860px);max-height:92vh"
    content-style="max-height:calc(92vh - 70px);overflow-y:auto">
    <div class="core-summary"><div><div class="core-summary-title">处理器核心概览</div><div class="core-summary-subtitle">每 5 秒刷新 · 点击卡片切换趋势分析</div></div><div class="core-summary-stats"><div class="core-summary-stat"><strong>{{ latestPerCore.length }}</strong><span>逻辑核心</span></div><div class="core-summary-divider"></div><div class="core-summary-stat"><strong>{{ averageCoreLoad.toFixed(1) }}%</strong><span>平均负载</span></div></div></div>
    <div class="core-grid">
      <button v-for="(value, index) in latestPerCore" :key="index" type="button" :class="['core-tile', { active: selectedCore === index }]" :style="{ '--core-color': coreLoadColor(value) }" @click="selectedCore = index">
        <div class="core-tile-head"><span class="core-index"><i></i>核心 {{ index + 1 }}</span><strong>{{ Number(value).toFixed(1) }}%</strong></div>
        <div class="core-meter"><span :style="{ width: Math.min(100, Number(value) || 0) + '%' }"></span></div>
      </button>
    </div>
    <div class="core-chart-panel">
      <div class="core-chart-head"><div><div class="core-chart-title">核心 {{ selectedCore + 1 }} 负载趋势</div><div class="core-chart-subtitle">最近约 5 分钟 · 固定 0–100% 量程</div></div><div class="core-current" :style="{ color: coreLoadColor(latestPerCore[selectedCore]) }">{{ Number(latestPerCore[selectedCore] || 0).toFixed(1) }}<small>%</small></div></div>
      <div class="core-chart-wrap">
      <svg viewBox="0 0 260 80" preserveAspectRatio="none">
        <polygon v-if="coreTrend.area" :points="coreTrend.area" fill="#4098fc" opacity="0.12" />
        <polyline v-for="(seg, idx) in coreTrend.segments" :key="idx" v-show="seg.length > 1" :points="seg.join(' ')" fill="none" stroke="#4098fc" stroke-width="2" />
      </svg>
      <n-empty v-if="!coreTrend.segments?.some(s => s.length > 1)" description="正在收集核心数据，请稍候" size="small" />
      </div>
    </div>
  </n-modal>

</template>

<style scoped lang="less">
.overview-panel { min-height:186px; }
.core-summary { display:flex; align-items:center; justify-content:space-between; gap:18px; padding:14px 16px; margin-bottom:14px; border-radius:12px; background:linear-gradient(135deg,rgba(64,152,252,.12),rgba(99,125,255,.04)); border:1px solid rgba(64,152,252,.18); }
.core-summary-title { font-size:14px; font-weight:800; }
.core-summary-subtitle,.core-chart-subtitle { margin-top:3px; font-size:11px; opacity:.55; }
.core-summary-stats { display:flex; align-items:center; gap:16px; }
.core-summary-stat { display:flex; flex-direction:column; align-items:flex-end; min-width:62px; }
.core-summary-stat strong { font-size:19px; line-height:1.1; color:#4098fc; }
.core-summary-stat span { margin-top:3px; font-size:10px; opacity:.55; }
.core-summary-divider { width:1px; height:30px; background:rgba(128,128,128,.2); }
.core-grid { display:grid; grid-template-columns:repeat(auto-fit,minmax(108px,1fr)); gap:8px; max-height:230px; overflow:auto; padding:2px 3px 8px 2px; }
.core-tile { --core-color:#4098fc; appearance:none; color:inherit; text-align:left; padding:10px 11px; border-radius:9px; border:1px solid rgba(128,128,128,.14); background:rgba(128,128,128,.045); cursor:pointer; transition:transform .16s ease,border-color .16s ease,background .16s ease,box-shadow .16s ease; }
.core-tile:hover { transform:translateY(-1px); border-color:var(--core-color); background:rgba(64,152,252,.08); }
.core-tile.active { border-color:var(--core-color); background:rgba(64,152,252,.12); box-shadow:0 0 0 2px rgba(64,152,252,.1); }
.core-tile-head { display:flex; align-items:center; justify-content:space-between; gap:6px; font-size:11px; }
.core-tile-head strong { font-size:12px; font-variant-numeric:tabular-nums; }
.core-index { display:flex; align-items:center; gap:6px; opacity:.7; white-space:nowrap; }
.core-index i { width:6px; height:6px; border-radius:50%; background:var(--core-color); box-shadow:0 0 7px var(--core-color); }
.core-meter { height:4px; margin-top:9px; overflow:hidden; border-radius:8px; background:rgba(128,128,128,.16); }
.core-meter span { display:block; height:100%; min-width:2px; border-radius:inherit; background:var(--core-color); box-shadow:0 0 8px var(--core-color); transition:width .4s ease; }
.core-chart-panel { margin-top:14px; padding:14px 16px 10px; border-radius:12px; border:1px solid rgba(128,128,128,.13); background:linear-gradient(180deg,rgba(128,128,128,.045),rgba(128,128,128,.015)); }
.core-chart-head { display:flex; align-items:center; justify-content:space-between; margin-bottom:8px; }
.core-chart-title { font-size:13px; font-weight:800; }
.core-current { font-size:24px; font-weight:800; line-height:1; font-variant-numeric:tabular-nums; }
.core-current small { margin-left:2px; font-size:12px; }
.core-chart-wrap { position:relative; min-height:112px; }
.core-chart-wrap svg { width:100%; height:112px; overflow:visible; }
@media (max-width:560px) { .core-summary { align-items:flex-start; flex-direction:column; } .core-summary-stats { width:100%; justify-content:flex-end; } .core-grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }

.server-mgmt { display: flex; flex-direction: column; gap: 0; }
.server-version-panel { padding: 9px 10px; border-radius: 8px; background: rgba(64,152,252,.07); border: 1px solid rgba(64,152,252,.16); }
.build-id-line { display: block; margin-top: 4px; font-family: monospace; font-size: 11px; }
.version-error { display: block; margin-top: 6px; font-size: 11px; word-break: break-word; }
.update-button-highlight { animation: update-pulse 1.5s ease-in-out infinite; box-shadow: 0 0 0 0 rgba(208,48,80,.35); }
@keyframes update-pulse { 50% { box-shadow: 0 0 0 5px rgba(208,48,80,.08); transform: translateY(-1px); } }
.mgmt-label { font-size: 13px; font-weight: 500; margin-bottom: 2px; }
.mgmt-desc { font-size: 11px; }

.plugin-card { padding: 2px 0; }
.plugin-icon { width: 28px; height: 28px; flex-shrink: 0; margin-top: 2px; }
.plugin-name { font-size: 13px; font-weight: 700; }
.plugin-desc { font-size: 12px; opacity: 0.65; margin: 4px 0 0; line-height: 1.4; }
.plugin-note { font-size: 11px; opacity: 0.5; margin: 5px 0 0; }
.plugin-badge {
  display: inline-flex; align-items: center; gap: 3px;
  padding: 2px 8px; border-radius: 20px; font-size: 11px; font-weight: 700;
  border: 1.5px solid;
  &.installed { border-color: rgba(24,160,88,.4); background: rgba(24,160,88,.12); color: #18a058; }
  &.not-installed { border-color: rgba(208,48,80,.3); background: rgba(208,48,80,.08); color: #d03050; }
  &.checking { border-color: rgba(64,152,252,.3); background: rgba(64,152,252,.08); color: #4098fc; }
.plugin-version {
  font-family: monospace;
  font-size: 10px;
  font-weight: 400;
  opacity: 0.75;
  margin-left: 4px;
}
}

.credits-block {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  padding-bottom: 8px;
}
.credits-inner {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-radius: 10px;
  background: rgba(128,128,128,.06);
  border: 1px solid rgba(128,128,128,.12);
  max-width: 420px;
}
.credits-logo {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(0,0,0,.15);
}
.credits-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.credits-made {
  display: flex;
  align-items: center;
  gap: 6px;
}
.credits-author {
  font-size: 14px;
  font-weight: 700;
  letter-spacing: 0.5px;
  cursor: default;
}
.credits-row {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
}
.credits-label {
  font-size: 11px;
  opacity: 0.5;
  white-space: nowrap;
}
.credits-link {
  font-size: 11px;
  opacity: 0.75;
  color: inherit;
  text-decoration: none;
  &:hover { opacity: 1; text-decoration: underline; }
}
.credits-sep {
  opacity: 0.35;
  font-size: 11px;
}
.mod-progress-log {
  max-height: 300px;
  overflow-y: auto;
  background: rgba(0,0,0,.15);
  border-radius: 6px;
  padding: 10px 12px;
  font-family: monospace;
  font-size: 12px;
}
.mod-log-line {
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
}
.mod-log-spinner {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  opacity: 0.7;
}
.mod-log-done {
  margin-top: 8px;
  color: #18a058;
  font-weight: 700;
}
.mod-log-failed {
  margin-top: 8px;
  color: #d03050;
  font-weight: 700;
}
</style>
