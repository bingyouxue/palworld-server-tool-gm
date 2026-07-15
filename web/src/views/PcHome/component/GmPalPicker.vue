<script setup>
import { ref, computed } from "vue";
import { useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import gmPalsRaw from "@/assets/gm/pals.json";
import palNameMap from "@/assets/pal.json";

const props = defineProps({
  playerUid: { type: String, required: true },
  onlinePlayers: { type: Array, default: () => [] },
});
const emit = defineEmits(["done"]);

const { locale } = useI18n();
const message = useMessage();

// ── 元素/工作图标 ────────────────────────────────────────
const ELEMENT_ICONS = {
  "无属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_00.webp",
  "火属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_01.webp",
  "水属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_02.webp",
  "雷属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_03.webp",
  "草属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_04.webp",
  "暗属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_05.webp",
  "龙属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_06.webp",
  "地属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_07.webp",
  "冰属性": "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_08.webp",
};

// ── 分类 ─────────────────────────────────────────────────
const CATEGORIES = [
  { key: "pal",       label: "普通" },
  { key: "boss",      label: "头目" },
  { key: "predator",  label: "狂暴化" },
  { key: "gym",       label: "高塔" },
  { key: "raid",      label: "突袭" },
  { key: "human",     label: "Human" },
];

// ── 合并帕鲁数据（gmPals有icon，palNameMap有中文名） ──────
const palZhMap = computed(() => {
  const lang = locale.value;
  return palNameMap[lang] || palNameMap["zh"] || {};
});

const allPals = computed(() =>
  gmPalsRaw.map((raw) => ({
    ...raw,
    label: palZhMap.value[raw.id] || raw.zh || raw.name || raw.id,
  })),
);

// ── 帕鲁图标：用 gmPals 的 icon 字段映射到 /gm-data/pals/ ─
const palIconMap = computed(() => {
  const m = {};
  for (const raw of gmPalsRaw) {
    if (raw.icon) m[raw.id] = `/gm-data/pals/${raw.icon}`;
  }
  return m;
});
const getPalAvatar = (id) => palIconMap.value[id] || "/gm-data/pals/T_icon_unknown.webp";

// ── 过滤 ─────────────────────────────────────────────────
const activeCategory = ref("pal");
const searchValue = ref("");

const filteredPals = computed(() => {
  const kw = searchValue.value.trim().toLowerCase();
  let list = allPals.value;
  if (kw) {
    list = list.filter(
      (p) =>
        (p.label || "").toLowerCase().includes(kw) ||
        (p.id || "").toLowerCase().includes(kw) ||
        (p.name || "").toLowerCase().includes(kw),
    );
  } else {
    list = list.filter((p) => (p.type || "pal") === activeCategory.value);
  }
  return list.slice(0, 300);
});

// ── 选择与给予 ────────────────────────────────────────────
const selectedPal = ref(null);
const givePalLevel = ref(1);
const giving = ref(false);

const handleGivePal = async () => {
  if (!selectedPal.value) { message.warning("请先选择帕鲁"); return; }
  giving.value = true;
  try {
    // givepal 命令通过 RCON 发送：givepal <userId> <palId> <level>
    const onlinePlayer = props.onlinePlayers.find(
      (p) => p.player_uid === props.playerUid || p.PlayerUid === props.playerUid,
    );
    const userId =
      onlinePlayer?.user_id ||
      onlinePlayer?.UserId ||
      "";
    if (!userId) {
      message.warning("玩家必须在线才能给予帕鲁（需要 user_id）");
      giving.value = false;
      return;
    }
    const cmd = `givepal ${userId} ${selectedPal.value.id} ${givePalLevel.value}`;
    const { data, statusCode } = await new ApiService().sendRconCommand({ command: cmd });
    if (statusCode.value === 200) {
      message.success(`已给予 ${selectedPal.value.label} Lv.${givePalLevel.value}`);
      emit("done");
    } else {
      message.error("给予失败: " + (data.value?.error || ""));
    }
  } catch (e) {
    message.error("给予失败: " + e.message);
  } finally {
    giving.value = false;
  }
};
</script>

<template>
  <div class="gm-pal-picker">
    <!-- 分类标签页 -->
    <div class="gm-category-tabs">
      <div
        v-for="cat in CATEGORIES"
        :key="cat.key"
        class="gm-cat-tab"
        :class="{ active: activeCategory === cat.key && !searchValue }"
        @click="activeCategory = cat.key; searchValue = ''"
      >{{ cat.label }}</div>
    </div>

    <!-- 搜索栏 -->
    <div class="gm-search-row">
      <n-input
        v-model:value="searchValue"
        clearable
        placeholder="搜索帕鲁名称 / ID（搜索时忽略分类）"
        size="small"
      >
        <template #prefix>
          <n-icon>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
              <path d="M221.09 64a157.09 157.09 0 1 0 157.09 157.09A157.1 157.1 0 0 0 221.09 64z" fill="none" stroke="currentColor" stroke-miterlimit="10" stroke-width="32"/>
              <path fill="none" stroke="currentColor" stroke-linecap="round" stroke-miterlimit="10" stroke-width="32" d="M338.29 338.29L448 448"/>
            </svg>
          </n-icon>
        </template>
      </n-input>
    </div>

    <!-- 帕鲁网格 -->
    <div class="gm-pal-grid">
      <div
        v-for="pal in filteredPals" :key="pal.id"
        class="gm-pal-card"
        :class="{ 'is-selected': selectedPal?.id === pal.id }"
        @click="selectedPal = pal"
      >
        <div class="gm-pal-img-wrap">
          <img
            :src="getPalAvatar(pal.id)"
            :alt="pal.label"
            class="gm-pal-img"
            @error="$event.target.style.display='none'"
          />
        </div>
        <div class="gm-pal-name">{{ pal.label }}</div>
        <div class="gm-pal-id">{{ pal.id }}</div>
      </div>
      <div v-if="filteredPals.length === 0" class="gm-empty">
        <n-empty description="无匹配结果" />
      </div>
    </div>

    <!-- 选中信息 + 给予操作 -->
    <div class="gm-give-bar">
      <div v-if="selectedPal" class="gm-selected-info">
        <img
          :src="getPalAvatar(selectedPal.id)"
          class="gm-sel-img"
          @error="$event.target.style.display='none'"
        />
        <div class="gm-sel-text">
          <span class="gm-sel-name">{{ selectedPal.label }}</span>
          <span class="gm-sel-id">{{ selectedPal.id }}</span>
        </div>
      </div>
      <div v-else class="gm-no-selection">请在上方点选一只帕鲁</div>
      <div class="gm-give-actions">
        <span class="gm-level-label">等级</span>
        <n-input-number
          v-model:value="givePalLevel"
          :min="1" :max="60"
          size="small"
          style="width: 90px"
        />
        <n-button
          type="primary" :loading="giving"
          :disabled="!selectedPal"
          round size="small"
          @click="handleGivePal"
        >给予帕鲁</n-button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="less">
.gm-pal-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.gm-category-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 0 2px;
}
.gm-cat-tab {
  padding: 3px 12px;
  border-radius: 20px;
  font-size: 12px;
  cursor: pointer;
  border: 1px solid rgba(64, 152, 252, 0.25);
  background: transparent;
  transition: all 0.15s;
  user-select: none;
  &:hover { border-color: #4098fc; background: rgba(64, 152, 252, 0.08); }
  &.active { border-color: #4098fc; background: #4098fc; color: #fff; }
}

.gm-search-row { padding: 0 2px; }

.gm-pal-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(90px, 1fr));
  gap: 6px;
  max-height: 360px;
  overflow-y: auto;
  padding: 2px;
}

.gm-pal-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px 6px;
  border: 1px solid rgba(24, 24, 28, 0.1);
  border-radius: 8px;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
  text-align: center;
  &:hover { border-color: #4098fc; background: rgba(64, 152, 252, 0.06); }
  &.is-selected {
    border-color: #4098fc;
    background: rgba(64, 152, 252, 0.14);
    box-shadow: 0 0 0 2px rgba(64, 152, 252, 0.3);
  }
}
.is-dark .gm-pal-card { border-color: rgba(255,255,255,0.1); }

.gm-pal-img-wrap {
  width: 52px; height: 52px;
  display: flex; align-items: center; justify-content: center;
}
.gm-pal-img {
  width: 52px; height: 52px;
  object-fit: contain;
}
.gm-pal-name {
  font-size: 11px; font-weight: 500;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden; text-overflow: ellipsis; width: 100%;
}
.gm-pal-id {
  font-size: 9px; opacity: 0.4;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; width: 100%;
}

.gm-empty { grid-column: 1 / -1; padding: 20px 0; }

.gm-give-bar {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px;
  background: rgba(64, 152, 252, 0.05);
  border: 1px solid rgba(64, 152, 252, 0.15);
  border-radius: 10px;
  flex-wrap: wrap;
}
.gm-selected-info {
  display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0;
}
.gm-sel-img {
  width: 40px; height: 40px; object-fit: contain; flex-shrink: 0;
}
.gm-sel-text { display: flex; flex-direction: column; min-width: 0; }
.gm-sel-name { font-size: 13px; font-weight: 600; }
.gm-sel-id { font-size: 11px; opacity: 0.5; }
.gm-no-selection { flex: 1; font-size: 12px; opacity: 0.5; }
.gm-give-actions {
  display: flex; align-items: center; gap: 8px; flex-shrink: 0;
}
.gm-level-label { font-size: 12px; }
</style>
