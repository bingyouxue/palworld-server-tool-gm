<script setup>
import { ref, computed } from "vue";
import { useMessage } from "naive-ui";
import ApiService from "@/service/api";

const props = defineProps({
  show: { type: Boolean, default: false },
  playerUid: { type: String, required: true },
  onlinePlayers: { type: Array, default: () => [] },
});
const emit = defineEmits(["update:show", "done"]);

const message = useMessage();

const tpMode = ref("coord"); // "coord" | "player"
const coordX = ref("");
const coordY = ref("");
const coordZ = ref("0");
const targetPlayerUserId = ref("");
const submitting = ref(false);

const close = () => emit("update:show", false);

// 当前玩家的 userId（需要在线才能传送）
const currentUserIdComputed = computed(() => {
  const online = props.onlinePlayers.find(
    (p) => (p.player_uid || p.PlayerUid) === props.playerUid,
  );
  return online?.user_id || online?.UserId || "";
});

const onlineOptions = computed(() =>
  props.onlinePlayers
    .filter((p) => (p.player_uid || p.PlayerUid) !== props.playerUid)
    .map((p) => ({
      label: `${p.nickname || p.name || "--"} (${p.user_id || p.UserId || ""})`,
      value: p.user_id || p.UserId || "",
    })),
);

const handleSubmit = async () => {
  const userId = currentUserIdComputed.value;
  if (!userId) {
    message.warning("该玩家当前不在线，无法传送");
    return;
  }

  let cmd = "";
  if (tpMode.value === "coord") {
    const x = coordX.value.trim();
    const y = coordY.value.trim();
    const z = coordZ.value.trim() || "0";
    if (!x || !y) { message.warning("请输入 X 和 Y 坐标"); return; }
    cmd = `tp ${userId} ${x} ${y} ${z}`;
  } else {
    if (!targetPlayerUserId.value) { message.warning("请选择目标玩家"); return; }
    cmd = `teleporttoplayer ${userId} ${targetPlayerUserId.value}`;
  }

  submitting.value = true;
  try {
    const { data, statusCode } = await new ApiService().sendRconCommand({ command: cmd });
    if (statusCode.value === 200) {
      message.success("传送指令已发送");
      emit("done");
      close();
    } else {
      message.error("传送失败: " + (data.value?.error || ""));
    }
  } catch (e) {
    message.error("传送失败: " + e.message);
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    style="width: 90%; max-width: 460px"
    title="传送此玩家"
    header-style="padding: 14px 20px;"
    content-style="padding: 16px 20px;"
    footer-style="padding: 12px 20px;"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <div class="tp-modal">
      <!-- 在线状态提示 -->
      <n-alert v-if="!currentUserIdComputed" type="warning" :show-icon="true" class="mb-4">
        该玩家当前不在线，传送命令需要玩家在线才能执行。
      </n-alert>

      <!-- 模式切换 -->
      <n-radio-group v-model:value="tpMode" class="mb-4">
        <n-radio-button value="coord">传送到坐标</n-radio-button>
        <n-radio-button value="player">传送到玩家</n-radio-button>
      </n-radio-group>

      <!-- 坐标模式 -->
      <div v-if="tpMode === 'coord'" class="tp-coord-form">
        <n-form label-placement="left" label-width="50" size="small">
          <n-form-item label="X">
            <n-input v-model:value="coordX" placeholder="如 -363823" clearable />
          </n-form-item>
          <n-form-item label="Y">
            <n-input v-model:value="coordY" placeholder="如 159033" clearable />
          </n-form-item>
          <n-form-item label="Z">
            <n-input v-model:value="coordZ" placeholder="高度（默认 0）" clearable />
          </n-form-item>
        </n-form>
        <div class="tp-hint">提示：坐标可从玩家详情页的 X / Y 字段复制</div>
      </div>

      <!-- 玩家模式 -->
      <div v-else class="tp-player-form">
        <n-select
          v-model:value="targetPlayerUserId"
          :options="onlineOptions"
          placeholder="选择目标在线玩家"
          filterable
          clearable
          size="small"
        />
        <div v-if="onlineOptions.length === 0" class="tp-hint mt-2">
          当前无其他在线玩家
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <n-button size="small" @click="close">取消</n-button>
        <n-button
          type="primary" size="small"
          :loading="submitting"
          :disabled="!currentUserIdComputed"
          @click="handleSubmit"
        >确认传送</n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped lang="less">
.tp-modal { display: flex; flex-direction: column; gap: 8px; }
.tp-coord-form, .tp-player-form { display: flex; flex-direction: column; }
.tp-hint { font-size: 11px; opacity: 0.5; margin-top: 4px; }
</style>
