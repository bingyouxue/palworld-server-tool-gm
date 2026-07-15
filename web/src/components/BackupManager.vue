<script setup>
import { ref, watch } from "vue";
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
// busyId tracks which backup_id has an in-flight action; suffix distinguishes action type
const busyId = ref("");

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
    // Use native fetch so we can get a proper Blob for file download
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
  (show) => show && load(),
  { immediate: true },
);
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    style="width: 94%; max-width: 820px"
    title="存档备份管理"
    @update:show="emit('update:show', $event)"
  >
    <n-spin :show="loading">
      <n-empty
        v-if="!loading && backups.length === 0"
        description="暂无备份记录"
      >
        <template #extra>
          <n-text depth="3">备份会在配置的时间间隔自动生成，也可手动触发同步。</n-text>
        </template>
      </n-empty>

      <n-list v-else hoverable bordered>
        <n-list-item v-for="backup in backups" :key="backup.backup_id">
          <n-thing>
            <template #header>
              <n-text strong>{{ dayjs(backup.save_time).format("YYYY-MM-DD HH:mm:ss") }}</n-text>
            </template>
            <template #description>
              <n-text depth="3" style="font-size:12px">{{ backup.path }}</n-text>
            </template>
          </n-thing>

          <template #suffix>
            <n-flex :size="6" align="center">
              <!-- 下载 -->
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

              <!-- 还原 -->
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

              <!-- 删除 -->
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
          </template>
        </n-list-item>
      </n-list>
    </n-spin>
  </n-modal>
</template>
