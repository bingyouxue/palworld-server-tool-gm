<script setup>
import { ref, computed, watch } from "vue";
import dayjs from "dayjs";
import { useMessage, useDialog } from "naive-ui";
import ApiService from "@/service/api";

const props = defineProps({ show: Boolean });
const emit = defineEmits(["update:show"]);
const message = useMessage();
const dialog = useDialog();
const api = new ApiService();

const backups = ref([]);
const loading = ref(false);
const busyId = ref("");

// ── 日历筛选 ──────────────────────────────────────────────────────────────
const showCalendar = ref(false);
const selectedDate = ref(null); // "YYYY-MM-DD" string or null

const backupDates = computed(() => {
  const s = new Set();
  for (const b of backups.value) {
    s.add(dayjs(b.save_time).format("YYYY-MM-DD"));
  }
  return s;
});

const filteredBackups = computed(() => {
  if (!selectedDate.value) return backups.value;
  return backups.value.filter(
    (b) => dayjs(b.save_time).format("YYYY-MM-DD") === selectedDate.value
  );
});

// 紧凑月历状态
const calYear = ref(dayjs().year());
const calMonth = ref(dayjs().month()); // 0-based

const calDays = computed(() => {
  const first = dayjs(new Date(calYear.value, calMonth.value, 1));
  const daysInMonth = first.daysInMonth();
  const startDow = first.day(); // 0=Sun
  const cells = [];
  for (let i = 0; i < startDow; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(d);
  return cells;
});

const calTitle = computed(() =>
  `${calYear.value} 年 ${calMonth.value + 1} 月`
);

const prevMonth = () => {
  if (calMonth.value === 0) { calMonth.value = 11; calYear.value--; }
  else calMonth.value--;
};
const nextMonth = () => {
  if (calMonth.value === 11) { calMonth.value = 0; calYear.value++; }
  else calMonth.value++;
};
const goToday = () => {
  calYear.value = dayjs().year();
  calMonth.value = dayjs().month();
};

const selectDay = (d) => {
  if (!d) return;
  const dateStr = `${calYear.value}-${String(calMonth.value + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  selectedDate.value = dateStr;
  showCalendar.value = false;
  checkedIds.value = [];
};

const isDaySelected = (d) => {
  if (!d || !selectedDate.value) return false;
  const dateStr = `${calYear.value}-${String(calMonth.value + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return dateStr === selectedDate.value;
};

const isDayToday = (d) => {
  if (!d) return false;
  const dateStr = `${calYear.value}-${String(calMonth.value + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return dateStr === dayjs().format("YYYY-MM-DD");
};

const isDayHasBackup = (d) => {
  if (!d) return false;
  const dateStr = `${calYear.value}-${String(calMonth.value + 1).padStart(2, "0")}-${String(d).padStart(2, "0")}`;
  return backupDates.value.has(dateStr);
};

const clearDateFilter = () => {
  selectedDate.value = null;
  checkedIds.value = [];
};

// ── 复选框 & 批量删除 ─────────────────────────────────────────────────────
const checkedIds = ref([]);

const allChecked = computed(() =>
  filteredBackups.value.length > 0 &&
  filteredBackups.value.every((b) => checkedIds.value.includes(b.backup_id))
);
const indeterminate = computed(() =>
  checkedIds.value.length > 0 && !allChecked.value &&
  filteredBackups.value.some((b) => checkedIds.value.includes(b.backup_id))
);

const toggleAll = (v) => {
  if (v) {
    checkedIds.value = filteredBackups.value.map((b) => b.backup_id);
  } else {
    checkedIds.value = [];
  }
};

const batchDeleting = ref(false);

const batchRemove = () => {
  if (checkedIds.value.length === 0) return;
  dialog.warning({
    title: "批量删除备份",
    content: `确定要删除已勾选的 ${checkedIds.value.length} 个备份吗？此操作不可恢复。`,
    positiveText: "全部删除",
    negativeText: "取消",
    onPositiveClick: async () => {
      batchDeleting.value = true;
      let ok = 0, fail = 0;
      for (const id of [...checkedIds.value]) {
        try {
          const res = await api.removeBackup(id);
          const code = res.statusCode?.value ?? res.statusCode;
          if (code === 200) ok++;
          else fail++;
        } catch {
          fail++;
        }
      }
      batchDeleting.value = false;
      checkedIds.value = [];
      if (ok > 0) message.success(`已删除 ${ok} 个备份`);
      if (fail > 0) message.error(`${fail} 个备份删除失败`);
      await load();
    },
  });
};

// ── 加载 ──────────────────────────────────────────────────────────────────
const load = async () => {
  loading.value = true;
  try {
    const res = await api.getBackupList({});
    const data = res.data?.value ?? res.data;
    backups.value = Array.isArray(data) ? data : [];
  } catch (e) {
    message.error("获取备份列表失败: " + e.message);
  } finally {
    loading.value = false;
  }
};

// ── 下载 ──────────────────────────────────────────────────────────────────
const download = async (backup) => {
  busyId.value = backup.backup_id + ":dl";
  try {
    const token = localStorage.getItem("palworld_token") || "";
    const resp = await fetch(`/api/backup/${backup.backup_id}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!resp.ok) {
      const body = await resp.json().catch(() => ({}));
      throw new Error(body?.error || `HTTP ${resp.status}`);
    }
    const blob = await resp.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = backup.path || `backup-${backup.backup_id}.zip`;
    a.click();
    URL.revokeObjectURL(url);
    message.success("下载成功");
  } catch (e) {
    message.error("下载失败: " + e.message);
  } finally {
    busyId.value = "";
  }
};

// ── 删除 ──────────────────────────────────────────────────────────────────
const remove = (backup) => {
  dialog.warning({
    title: "删除备份",
    content: `确定要删除备份 "${dayjs(backup.save_time).format("YYYY-MM-DD HH:mm:ss")}" 吗？此操作不可恢复。`,
    positiveText: "删除",
    negativeText: "取消",
    onPositiveClick: async () => {
      busyId.value = backup.backup_id + ":del";
      try {
        const res = await api.removeBackup(backup.backup_id);
        const code = res.statusCode?.value ?? res.statusCode;
        const data = res.data?.value ?? res.data;
        if (code === 200) {
          message.success("备份已删除");
          checkedIds.value = checkedIds.value.filter((id) => id !== backup.backup_id);
          await load();
        } else {
          message.error("删除失败: " + (data?.error || "未知错误"));
        }
      } catch (e) {
        message.error("删除失败: " + e.message);
      } finally {
        busyId.value = "";
      }
    },
  });
};

// ── 还原 ──────────────────────────────────────────────────────────────────
const restore = (backup) => {
  dialog.warning({
    title: "还原存档备份",
    content: `还原操作将：\n① 向游戏服务器发送关机指令\n② 等待服务器停止（约 15 秒）\n③ 将备份 "${dayjs(backup.save_time).format("YYYY-MM-DD HH:mm:ss")}" 解压覆盖到当前存档目录\n④ 自动重启服务器\n\n当前在线玩家的进度将丢失，确定继续？`,
    positiveText: "确认还原",
    negativeText: "取消",
    type: "warning",
    onPositiveClick: async () => {
      busyId.value = backup.backup_id + ":restore";
      message.loading("正在还原存档，请稍候（约 20 秒）…", { duration: 0, key: "restore-progress" });
      try {
        const res = await api.restoreBackup(backup.backup_id);
        const code = res.statusCode?.value ?? res.statusCode;
        const data = res.data?.value ?? res.data;
        message.destroyAll();
        if (code === 200) {
          message.success(data?.message || "存档还原成功，服务器正在重启", { duration: 6000 });
        } else {
          message.error("还原失败: " + (data?.error || data?.message || "未知错误"), { duration: 8000 });
        }
      } catch (e) {
        message.destroyAll();
        message.error("还原失败: " + e.message, { duration: 8000 });
      } finally {
        busyId.value = "";
      }
    },
  });
};

watch(
  () => props.show,
  (show) => {
    if (show) {
      load();
      checkedIds.value = [];
      selectedDate.value = null;
      calYear.value = dayjs().year();
      calMonth.value = dayjs().month();
    }
  },
  { immediate: true },
);
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    style="width: 94%; max-width: 860px"
    title="存档备份管理"
    @update:show="emit('update:show', $event)"
  >
    <n-spin :show="loading">
      <!-- 工具栏 -->
      <div class="bm-toolbar">
        <!-- 左侧：日历筛选 -->
        <div class="bm-toolbar-left">
          <n-popover
            v-model:show="showCalendar"
            trigger="click"
            placement="bottom-start"
            :width="280"
            :padding="0"
          >
            <template #trigger>
              <n-button size="small" secondary>
                <template #icon>
                  <n-icon>
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                      <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
                      <line x1="16" y1="2" x2="16" y2="6"/>
                      <line x1="8" y1="2" x2="8" y2="6"/>
                      <line x1="3" y1="10" x2="21" y2="10"/>
                    </svg>
                  </n-icon>
                </template>
                {{ selectedDate ? selectedDate : "按日期筛选" }}
              </n-button>
            </template>

            <!-- 自制紧凑月历 -->
            <div class="bm-cal-wrap">
              <!-- 月份导航 -->
              <div class="bm-cal-header">
                <button class="bm-cal-nav" @click.stop="prevMonth">&#8249;</button>
                <span class="bm-cal-title">{{ calTitle }}</span>
                <button class="bm-cal-nav bm-cal-today" @click.stop="goToday">今</button>
                <button class="bm-cal-nav" @click.stop="nextMonth">&#8250;</button>
              </div>
              <!-- 星期头 -->
              <div class="bm-cal-grid">
                <div v-for="w in ['日','一','二','三','四','五','六']" :key="w" class="bm-cal-dow">{{ w }}</div>
                <!-- 日期格 -->
                <div
                  v-for="(d, i) in calDays"
                  :key="i"
                  class="bm-cal-day"
                  :class="{
                    'is-empty': !d,
                    'is-today': isDayToday(d),
                    'is-selected': isDaySelected(d),
                    'has-backup': isDayHasBackup(d),
                  }"
                  @click.stop="selectDay(d)"
                >
                  <span v-if="d">{{ d }}</span>
                  <i v-if="isDayHasBackup(d)" class="bm-cal-badge" />
                </div>
              </div>
            </div>
          </n-popover>

          <n-button
            v-if="selectedDate"
            size="small"
            quaternary
            type="primary"
            @click="clearDateFilter"
          >
            清除筛选
          </n-button>

          <n-text v-if="selectedDate" depth="3" style="font-size:12px">
            共 {{ filteredBackups.length }} 条
          </n-text>
        </div>

        <!-- 右侧：批量删除 -->
        <div class="bm-toolbar-right">
          <template v-if="checkedIds.length > 0">
            <n-text depth="3" style="font-size:12px">已选 {{ checkedIds.length }} 项</n-text>
            <n-button
              size="small"
              type="error"
              secondary
              :loading="batchDeleting"
              @click="batchRemove"
            >
              删除所选
            </n-button>
          </template>
        </div>
      </div>

      <!-- 空状态 -->
      <n-empty
        v-if="!loading && backups.length === 0"
        description="暂无备份记录"
        style="margin-top:16px"
      >
        <template #extra>
          <n-text depth="3">备份会在配置的时间间隔自动生成，也可手动触发同步。</n-text>
        </template>
      </n-empty>

      <n-empty
        v-else-if="!loading && filteredBackups.length === 0 && selectedDate"
        description="该日期无备份记录"
        style="margin-top:16px"
      />

      <template v-else-if="filteredBackups.length > 0">
        <!-- 全选行 -->
        <div class="bm-select-all-row">
          <n-checkbox
            :checked="allChecked"
            :indeterminate="indeterminate"
            @update:checked="toggleAll"
          >
            全选
          </n-checkbox>
        </div>

        <n-list hoverable bordered style="margin-top:4px">
          <n-list-item v-for="backup in filteredBackups" :key="backup.backup_id">
            <div class="bm-list-row">
              <!-- 复选框 -->
              <n-checkbox
                :checked="checkedIds.includes(backup.backup_id)"
                @update:checked="(v) => v
                  ? checkedIds.push(backup.backup_id)
                  : (checkedIds = checkedIds.filter(id => id !== backup.backup_id))"
              />

              <!-- 信息 -->
              <div class="bm-list-info">
                <n-text strong>{{ dayjs(backup.save_time).format("YYYY-MM-DD HH:mm:ss") }}</n-text>
                <n-text depth="3" style="font-size:12px">{{ backup.path }}</n-text>
              </div>

              <!-- 操作按钮 -->
              <n-flex :size="6" align="center" style="flex-shrink:0">
                <n-button
                  size="small"
                  type="primary"
                  secondary
                  :loading="busyId === backup.backup_id + ':dl'"
                  :disabled="!!busyId && busyId !== backup.backup_id + ':dl'"
                  @click="download(backup)"
                >
                  下载
                </n-button>
                <n-button
                  size="small"
                  type="warning"
                  secondary
                  :loading="busyId === backup.backup_id + ':restore'"
                  :disabled="!!busyId && busyId !== backup.backup_id + ':restore'"
                  @click="restore(backup)"
                >
                  还原
                </n-button>
                <n-button
                  size="small"
                  type="error"
                  secondary
                  :loading="busyId === backup.backup_id + ':del'"
                  :disabled="!!busyId && busyId !== backup.backup_id + ':del'"
                  @click="remove(backup)"
                >
                  删除
                </n-button>
              </n-flex>
            </div>
          </n-list-item>
        </n-list>
      </template>
    </n-spin>
  </n-modal>
</template>

<style scoped>
.bm-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  flex-wrap: wrap;
  gap: 8px;
}
.bm-toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.bm-toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.bm-select-all-row {
  padding: 4px 12px 4px;
  border-bottom: 1px solid rgba(128,128,128,0.15);
}
.bm-list-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
}
.bm-list-info {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  gap: 2px;
}
/* 自制紧凑月历 */
.bm-cal-wrap {
  padding: 10px;
  width: 280px;
  box-sizing: border-box;
  user-select: none;
}
.bm-cal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}
.bm-cal-title {
  font-size: 13px;
  font-weight: 600;
  flex: 1;
  text-align: center;
}
.bm-cal-nav {
  background: none;
  border: none;
  cursor: pointer;
  color: inherit;
  font-size: 16px;
  padding: 2px 6px;
  border-radius: 4px;
  line-height: 1;
  opacity: 0.65;
  transition: opacity 0.15s, background 0.15s;
}
.bm-cal-nav:hover { opacity: 1; background: rgba(128,128,128,0.12); }
.bm-cal-today { font-size: 11px; padding: 2px 5px; }
.bm-cal-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
}
.bm-cal-dow {
  text-align: center;
  font-size: 11px;
  opacity: 0.45;
  padding: 2px 0 4px;
}
.bm-cal-day {
  position: relative;
  text-align: center;
  font-size: 12px;
  height: 30px;
  line-height: 30px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.12s, color 0.12s;
}
.bm-cal-day:not(.is-empty):hover { background: rgba(64,152,252,0.15); }
.bm-cal-day.is-empty { cursor: default; }
.bm-cal-day.is-today {
  color: #4098fc;
  font-weight: 700;
}
.bm-cal-day.is-selected {
  background: #4098fc;
  color: #fff;
  font-weight: 600;
}
.bm-cal-day.is-selected:hover { background: #3080e0; }
.bm-cal-badge {
  display: block;
  position: absolute;
  bottom: 3px;
  left: 50%;
  transform: translateX(-50%);
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: #4098fc;
}
.bm-cal-day.is-selected .bm-cal-badge { background: rgba(255,255,255,0.8); }
</style>
