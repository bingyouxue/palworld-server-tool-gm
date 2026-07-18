<script setup>
import { ref, computed, onMounted } from "vue";
import { useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import gmItemsRaw from "@/assets/gm/gmItems.json";
import palItems from "@/assets/items.json";

const props = defineProps({
  playerUid: { type: String, required: true },
});
const emit = defineEmits(["done"]);

const { locale } = useI18n();
const message = useMessage();

// ── 分类映射（从icon前缀推断） ──────────────────────────
const CATEGORY_MAP = {
  Weapon:   "武器",
  Armor:    "防具",
  Accessory:"饰品",
  Food:     "食物",
  food:     "食物",
  Material: "材料",
  Consume:  "消耗品",
  Essential:"鞍具/道具书",
  Ammo:     "弹药",
  PalSphere:"帕鲁球",
  Blueprint:"设计图",
  Relic:    "遗迹碎片",
  QuestItem:"任务道具",
  Glider:   "滑翔伞",
  PalUpgradeStone: "强化石",
  SphereModule: "球模块",
  PalAwakening: "觉醒素材",
  Salvage:  "打捞",
  T:        "BOSS奖励",
  Tex:      "其他",
  Jewelry:  "票券",
};

const RARITY_LABELS = { 0: "常见", 1: "少见", 2: "稀有", 3: "史诗", 4: "传奇" };
const RARITY_COLORS = { 0: "default", 1: "success", 2: "info", 3: "default", 4: "warning" };

// 从 id 后缀推断品质：_Default1/无后缀=0 普通，_2=1 少见，_3=2 稀有，_4=3 史诗，_5=4 传奇
function getRarityFromId(id = "") {
  const m = id.match(/_Default(\d)$/) || id.match(/_(\d)$/)
  if (m) return Math.min(parseInt(m[1]) - 1, 4)
  return null
}

// 从icon路径提取分类前缀
function getCategoryFromIcon(icon = "", id = "") {
  if (icon === "T_itemicon_Material_Blueprint.webp" || id.startsWith("Blueprint_")) {
    return "设计图";
  }
  const clean = icon.replace("T_itemicon_", "").replace(/\.webp$/, "");
  const prefix = clean.split("_")[0];
  return CATEGORY_MAP[prefix] || "其他";
}

// ── 合并 gmItems（有icon+rarity）与 items.json（有中文名） ──
const allItems = computed(() => {
  const langMap = {};
  const lang = locale.value;
  const list = palItems[lang] || palItems["zh"] || [];
  for (const item of list) {
    langMap[item.id?.toLowerCase()] = item;
  }
  return gmItemsRaw.map((raw) => {
    const nameEntry = langMap[raw.id?.toLowerCase()];
    const idRarity = getRarityFromId(raw.id || "");
    return {
      ...raw,
      rarity: idRarity !== null ? idRarity : (raw.rarity ?? 0),
      label: nameEntry?.name || raw.zh || raw.name || raw.id,
      description: nameEntry?.description || "",
      category: getCategoryFromIcon(raw.icon || "", raw.id || ""),
    };
  }).filter((x) => x.label && x.label !== "-");
});

// ── 分类列表 ────────────────────────────────────────────
const categories = computed(() => {
  const map = new Map();
  map.set("全部", 0);
  for (const item of allItems.value) {
    map.set(item.category, (map.get(item.category) || 0) + 1);
  }
  map.set("全部", allItems.value.length);
  const order = ["全部","武器","防具","饰品","食物","帕鲁球","弹药","鞍具/道具书","消耗品","材料","设计图","蓝图","强化石","觉醒素材","遗迹碎片","其他"];
  const result = [];
  for (const k of order) {
    if (map.has(k)) result.push({ name: k, count: map.get(k) });
  }
  for (const [k, v] of map) {
    if (!order.includes(k)) result.push({ name: k, count: v });
  }
  return result;
});

const activeCategory = ref("全部");
const activeRarity = ref(null);
const searchValue = ref("");

const filteredItems = computed(() => {
  let list = allItems.value;
  if (activeCategory.value !== "全部") {
    list = list.filter((x) => x.category === activeCategory.value);
  }
  if (activeRarity.value !== null) {
    list = list.filter((x) => (x.rarity ?? 0) === activeRarity.value);
  }
  const kw = searchValue.value.trim().toLowerCase();
  if (kw) {
    list = list.filter(
      (x) =>
        (x.label || "").toLowerCase().includes(kw) ||
        (x.id || "").toLowerCase().includes(kw) ||
        (x.description || "").toLowerCase().includes(kw),
    );
  }
  return list.slice(0, 200);
});

// ── 图片：用 icon 字段映射到 public/gm-data/items ───────
const itemIconMap = computed(() => {
  const m = {};
  for (const raw of gmItemsRaw) {
    if (raw.icon) m[raw.id] = `/gm-data/items/${raw.icon}`;
  }
  return m;
});
const getItemIcon = (id) => itemIconMap.value[id] || "";

// ── 设计图叠加图标：从 Blueprint_XXX_N 推断出 XXX 对应的物品图标 ──
const blueprintOverlayMap = computed(() => {
  // baseId（小写）-> icon url
  const base2icon = {};
  for (const raw of gmItemsRaw) {
    if (!raw.id.startsWith("Blueprint_") && raw.icon && raw.icon !== "T_itemicon_Material_Blueprint.webp") {
      // 取物品 ID 去掉末尾 _DefaultN 或 _N 后的 base
      const base = raw.id.replace(/_Default\d$/, "").replace(/_\d$/, "").toLowerCase();
      if (!base2icon[base]) {
        base2icon[base] = `/gm-data/items/${raw.icon}`;
      }
    }
  }
  // 为每个设计图建立 id -> overlay icon
  const m = {};
  for (const raw of gmItemsRaw) {
    if (!raw.id.startsWith("Blueprint_")) continue;
    const base = raw.id
      .replace(/^Blueprint_/, "")
      .replace(/_Default\d$/, "")
      .replace(/_\d$/, "")
      .toLowerCase();
    if (base2icon[base]) m[raw.id] = base2icon[base];
  }
  return m;
});
const getBlueprintOverlay = (id) => blueprintOverlayMap.value[id] || "";

// ── 选择与给予 ───────────────────────────────────────────
const selectedItem = ref(null);
const giveAmount = ref(1);
const giving = ref(false);

const handleGive = async () => {
  if (!selectedItem.value) { message.warning("请先选择道具"); return; }
  giving.value = true;
  try {
    const { data, statusCode } = await new ApiService().giveItem({
      playerUid: props.playerUid,
      item_id: selectedItem.value.id,
      amount: giveAmount.value,
    });
    if (statusCode.value === 200) {
      message.success("道具已发送");
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
  <div class="gm-item-picker">
    <!-- 搜索栏 -->
    <div class="gm-search-row">
      <n-input
        v-model:value="searchValue"
        clearable
        placeholder="搜索道具名称 / ID"
        size="small"
      >
        <template #prefix><n-icon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path d="M221.09 64a157.09 157.09 0 1 0 157.09 157.09A157.1 157.1 0 0 0 221.09 64z" fill="none" stroke="currentColor" stroke-miterlimit="10" stroke-width="32"/><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-miterlimit="10" stroke-width="32" d="M338.29 338.29L448 448"/></svg></n-icon></template>
      </n-input>
    </div>

    <!-- 分类栏 -->
    <div class="gm-category-bar">
      <button
        v-for="cat in categories" :key="cat.name"
        class="gm-cat-chip" :class="{ active: activeCategory === cat.name }"
        type="button"
        @click="activeCategory = cat.name; selectedItem = null; activeTier = null"
      >
        {{ cat.name }}
        <span v-if="cat.name !== '全部'" class="gm-cat-count">{{ cat.count }}</span>
      </button>
    </div>

    <!-- 品质筛选栏 -->
    <div class="gm-rarity-bar">
      <button
        class="gm-rarity-chip" :class="{ active: activeRarity === null }"
        type="button"
        @click="activeRarity = null"
      >全部品质</button>
      <button
        v-for="(label, r) in RARITY_LABELS" :key="r"
        class="gm-rarity-chip" :class="[`rarity-${r}`, { active: activeRarity === Number(r) }]"
        type="button"
        @click="activeRarity = Number(r)"
      >{{ label }}</button>
    </div>

    <!-- 道具网格 -->
    <div class="gm-item-grid">
      <div
        v-for="item in filteredItems" :key="item.id"
        class="gm-item-card"
        :class="[`rarity-bg-${item.rarity ?? 0}`, { 'is-selected': selectedItem?.id === item.id }]"
        @click="selectedItem = item"
      >
        <div class="gm-item-img-wrap">
          <template v-if="getBlueprintOverlay(item.id)">
            <img :src="getItemIcon(item.id)" :alt="item.label" class="gm-item-img gm-blueprint-bg" @error="$event.target.style.display='none'" />
            <img :src="getBlueprintOverlay(item.id)" :alt="item.label" class="gm-item-img gm-blueprint-overlay" @error="$event.target.style.display='none'" />
          </template>
          <img
            v-else
            :src="getItemIcon(item.id)"
            :alt="item.label"
            class="gm-item-img"
            @error="$event.target.style.display='none'"
          />
        </div>
        <div class="gm-item-name">{{ item.label }}</div>
        <div class="gm-item-id">{{ item.id }}</div>
        <!-- 有等阶时显示等阶标签，否则显示品质标签 -->
        <n-tag v-if="item.rarity !== undefined" size="tiny" :type="RARITY_COLORS[item.rarity ?? 0]" :bordered="false" :class="['gm-item-rarity', { 'rarity-epic': (item.rarity ?? 0) === 3 }]">
          {{ RARITY_LABELS[item.rarity ?? 0] }}
        </n-tag>
      </div>
      <div v-if="filteredItems.length === 0" class="gm-empty">
        <n-empty description="无匹配结果" />
      </div>
    </div>

    <!-- 选中信息 + 操作区 -->
    <div class="gm-give-bar">
      <div v-if="selectedItem" class="gm-selected-info">
        <div class="gm-sel-img-wrap">
          <template v-if="getBlueprintOverlay(selectedItem.id)">
            <img :src="getItemIcon(selectedItem.id)" class="gm-sel-img gm-blueprint-bg" @error="$event.target.style.display='none'" />
            <img :src="getBlueprintOverlay(selectedItem.id)" class="gm-sel-img gm-sel-blueprint-overlay" @error="$event.target.style.display='none'" />
          </template>
          <img v-else :src="getItemIcon(selectedItem.id)" class="gm-sel-img" @error="$event.target.style.display='none'" />
        </div>
        <div class="gm-sel-text">
          <span class="gm-sel-name">{{ selectedItem.label }}</span>
          <span class="gm-sel-id">{{ selectedItem.id }}</span>
        </div>
      </div>
      <div v-else class="gm-no-selection">请在上方点选一个道具</div>
      <div class="gm-give-actions">
        <n-input-number v-model:value="giveAmount" :min="1" :max="9999" size="small" style="width:110px" />
        <n-button
          type="primary" :loading="giving"
          :disabled="!selectedItem"
          round size="small"
          @click="handleGive"
        >给予道具</n-button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="less">
.gm-item-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.gm-search-row {
  padding: 0 2px;
}

.gm-category-bar,
.gm-rarity-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 0 2px;
}

.gm-cat-chip,
.gm-rarity-chip {
  padding: 3px 10px;
  border-radius: 20px;
  font-size: 12px;
  border: 1px solid rgba(24, 24, 28, 0.12);
  background: transparent;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  color: inherit;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  &:hover { border-color: #4098fc; color: #4098fc; }
  &.active { background: #4098fc; border-color: #4098fc; color: #fff; }
}
.is-dark .gm-cat-chip,
.is-dark .gm-rarity-chip {
  border-color: rgba(255,255,255,0.15);
}
.gm-cat-count { font-size: 10px; opacity: 0.65; }

.gm-rarity-chip {
  &.rarity-1.active { background: #18a058; border-color: #18a058; }
  &.rarity-2.active { background: #4098fc; border-color: #4098fc; }
  &.rarity-3.active { background: #9d60f0; border-color: #9d60f0; color: #fff; }
  &.rarity-4.active { background: #f0a020; border-color: #f0a020; color: #fff; }
}

.gm-item-rarity.rarity-epic {
  background-color: rgba(157, 96, 240, 0.16) !important;
  color: #b47ef5 !important;
  border-color: rgba(157, 96, 240, 0.3) !important;
}

.gm-item-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 6px;
  max-height: 340px;
  overflow-y: auto;
  padding: 2px;
}

.gm-item-card {
  position: relative;
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
.is-dark .gm-item-card { border-color: rgba(255,255,255,0.1); }

.rarity-bg-0 { border-color: rgba(255,255,255,0.24); }
.rarity-bg-1 { border-color: rgba(24,160,88,0.25); }
.rarity-bg-2 { border-color: rgba(64,152,252,0.25); }
.rarity-bg-3 { border-color: rgba(157,96,240,0.3); }
.rarity-bg-4 { border-color: rgba(242,160,32,0.3); }

.gm-item-img-wrap {
  width: 48px; height: 48px;
  position: relative;
  display: flex; align-items: center; justify-content: center;
}
.gm-item-img {
  width: 48px; height: 48px;
  object-fit: contain;
}
.gm-blueprint-bg {
  position: absolute; top: 0; left: 0;
  width: 48px; height: 48px;
  object-fit: contain;
}
.gm-blueprint-overlay {
  position: absolute; top: 0; left: 0;
  width: 48px; height: 48px;
  object-fit: contain;
  transform: scale(0.64);
}
.gm-item-name {
  font-size: 11px; font-weight: 500;
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 100%;
}
.gm-item-id {
  font-size: 9px; opacity: 0.45;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; width: 100%;
}
.gm-item-rarity {
  position: absolute; top: 4px; right: 4px;
}

.gm-empty { grid-column: 1 / -1; padding: 20px 0; }

.gm-give-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: rgba(64, 152, 252, 0.05);
  border: 1px solid rgba(64, 152, 252, 0.15);
  border-radius: 10px;
  flex-wrap: wrap;
}
.gm-selected-info {
  display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0;
}
.gm-sel-img-wrap {
  position: relative; width: 36px; height: 36px; flex-shrink: 0;
}
.gm-sel-img {
  width: 36px; height: 36px; object-fit: contain; flex-shrink: 0;
}
.gm-sel-blueprint-overlay {
  position: absolute; top: 0; left: 0;
  width: 36px; height: 36px;
  object-fit: contain;
  transform: scale(0.64);
}
.gm-sel-text {
  display: flex; flex-direction: column; min-width: 0;
}
.gm-sel-name { font-size: 13px; font-weight: 600; }
.gm-sel-id { font-size: 11px; opacity: 0.5; }
.gm-no-selection { flex: 1; font-size: 12px; opacity: 0.5; }
.gm-give-actions {
  display: flex; align-items: center; gap: 8px; flex-shrink: 0;
}
</style>
