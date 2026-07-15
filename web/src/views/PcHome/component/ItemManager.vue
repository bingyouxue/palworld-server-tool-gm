<script setup>
import { computed, h, inject, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { useMessage, useDialog, NAvatar, NTag, NButton, NIcon } from "naive-ui";
import { ChevronForward } from "@vicons/ionicons5";
import ApiService from "@/service/api";
import palMap from "@/assets/pal.json";
import skillMap from "@/assets/skill.json";
import palItems from "@/assets/items.json";
import { localizedSkillName } from "@/utils/gameLabels";
import pageStore from "@/stores/model/page.js";

const { t, locale } = useI18n();
const message = useMessage();
const dialog = useDialog();

const props = defineProps({
  players: { type: Array, default: () => [] },
});

const isDarkMode = inject("isDarkMode", ref(window.matchMedia("(prefers-color-scheme: dark)").matches));
const pageWidth = computed(() => pageStore().getScreenWidth());
const smallScreen = computed(() => pageWidth.value < 1024);

const playerList = ref([]);
const selectedPlayer = ref(null);
const loadingDetail = ref(false);
const playerInfo = ref(null);
const searchPlayerValue = ref("");

const filteredPlayers = computed(() => {
  const kw = searchPlayerValue.value.trim().toLowerCase();
  if (!kw) return playerList.value;
  return playerList.value.filter((p) =>
    [p.nickname, p.player_uid, p.user_id, p.steam_id]
      .filter(Boolean).join(" ").toLowerCase().includes(kw),
  );
});

const selectPlayer = async (player) => {
  if (selectedPlayer.value?.player_uid === player.player_uid) return;
  selectedPlayer.value = player;
  loadingDetail.value = true;
  try {
    const { data } = await new ApiService().getPlayer({ playerUid: player.player_uid });
    playerInfo.value = data.value;
    mergeItems();
  } finally {
    loadingDetail.value = false;
  }
};

const activeTab = ref("give");

const itemSearchValue = ref("");
const allItems = computed(() => palItems[locale.value] || palItems["zh"] || []);
const filteredItems = computed(() => {
  const kw = itemSearchValue.value.trim().toLowerCase();
  if (!kw) return allItems.value.slice(0, 80);
  return allItems.value.filter(
    (item) =>
      (item.name || "").toLowerCase().includes(kw) ||
      (item.id || "").toLowerCase().includes(kw),
  );
});
const selectedItem = ref(null);
const giveAmount = ref(1);
const givingItem = ref(false);

const handleGiveItem = async () => {
  if (!selectedPlayer.value) { message.warning(t("message.selectPlayerFirst")); return; }
  if (!selectedItem.value) { message.warning(t("message.selectItemFirst")); return; }
  givingItem.value = true;
  try {
    const { data, statusCode } = await new ApiService().giveItem({
      playerUid: selectedPlayer.value.player_uid,
      item_id: selectedItem.value.id,
      amount: giveAmount.value,
    });
    if (statusCode.value === 200) {
      message.success(t("message.giveItemSuccess"));
    } else {
      message.error(t("message.giveItemFail", { err: data.value?.error }));
    }
  } catch (e) {
    message.error(t("message.giveItemFail", { err: e.message }));
  } finally {
    givingItem.value = false;
  }
};

const containerTab = ref("CommonContainerId");
const mergedItems = ref({});
const deletingItem = ref(false);

const mergeItems = () => {
  mergedItems.value = {};
  if (!playerInfo.value?.items) return;
  const lang = locale.value;
  for (const [cid, items] of Object.entries(playerInfo.value.items)) {
    mergedItems.value[cid] = items.map((item) => {
      const front = (palItems[lang] || palItems["zh"] || []).find((f) => f.id === item.ItemId);
      return front
        ? { ...item, id: front.id, name: front.name, description: front.description }
        : { ...item, id: item.ItemId, name: item.ItemId, description: "" };
    });
  }
};

const handleDeleteItem = (item) => {
  if (!selectedPlayer.value) return;
  dialog.warning({
    title: t("button.deleteItem"),
    content: `${t("button.deleteItem")} ${item.name} x${item.StackCount}？`,
    positiveText: t("button.confirmDelete"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      deletingItem.value = true;
      try {
        const { data, statusCode } = await new ApiService().deleteItem({
          playerUid: selectedPlayer.value.player_uid,
          item_id: item.id,
          amount: item.StackCount,
        });
        if (statusCode.value === 200) {
          message.success(t("message.deleteItemSuccess"));
          await refreshDetail();
        } else {
          message.error(t("message.deleteItemFail", { err: data.value?.error }));
        }
      } catch (e) {
        message.error(t("message.deleteItemFail", { err: e.message }));
      } finally {
        deletingItem.value = false;
      }
    },
  });
};

const palSearchValue = ref("");
const localeLowerPalMap = ref({});
const releasingPal = ref(false);

const filteredPals = computed(() => {
  const list = playerInfo.value?.pals || [];
  const kw = palSearchValue.value.trim().toLowerCase();
  if (!kw) return list;
  return list.filter(
    (p) => getPalName(p.type).toLowerCase().includes(kw) || p.type.toLowerCase().includes(kw),
  );
});

const getPalName = (type) => localeLowerPalMap.value[type.toLowerCase()] || type;

const getPalAvatar = (type) => {
  try {
    return new URL(`../../assets/pals/${type.toLowerCase()}.png`, import.meta.url).href;
  } catch {
    return getUnknownAvatar(false);
  }
};

const getUnknownAvatar = (isBoss = false) =>
  isBoss
    ? new URL("@/assets/pals/boss_unknown.png", import.meta.url).href
    : new URL("@/assets/pals/unknown.png", import.meta.url).href;

const handleReleasePal = (pal) => {
  if (!selectedPlayer.value) return;
  const name = getPalName(pal.type);
  dialog.warning({
    title: t("button.releasePal"),
    content: t("message.releasePalConfirm", { name, level: pal.level || "?" }),
    positiveText: t("button.confirmRelease"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      releasingPal.value = true;
      try {
        const { data, statusCode } = await new ApiService().releasePal({
          playerUid: selectedPlayer.value.player_uid,
          pal_type: pal.type,
          level: pal.level || 0,
          gender: pal.gender || "",
          is_lucky: pal.is_lucky || false,
        });
        if (statusCode.value === 200) {
          message.success(t("message.releasePalSuccess"));
          await refreshDetail();
        } else {
          message.error(t("message.releasePalFail", { err: data.value?.error }));
        }
      } catch (e) {
        message.error(t("message.releasePalFail", { err: e.message }));
      } finally {
        releasingPal.value = false;
      }
    },
  });
};

const refreshDetail = async () => {
  if (!selectedPlayer.value) return;
  const { data } = await new ApiService().getPlayer({ playerUid: selectedPlayer.value.player_uid });
  playerInfo.value = data.value;
  mergeItems();
};

const syncing = ref(false);
const syncAndRefresh = async () => {
  if (!selectedPlayer.value) return;
  syncing.value = true;
  try {
    await new ApiService().syncData("sav");
        await new Promise((r) => setTimeout(r, 3000));
    await refreshDetail();
    message.success(t("message.syncSuccess") || "存档同步完成，物品数据已更新");
  } catch (e) {
    message.error(e?.message || "同步失败");
  } finally {
    syncing.value = false;
  }
};

const getItemIcon = (id) => {
  try {
    return new URL(`../../assets/items/${id}.webp`, import.meta.url).href;
  } catch {
    return getUnknownAvatar();
  }
};

const createItemColumns = () => [
  {
    title: "", key: "_icon", width: 48,
    render: (row) => h(NAvatar, { size: "small", src: getItemIcon(row.id), fallbackSrc: getUnknownAvatar() }),
  },
  { title: t("item.name"), key: "name" },
  { title: t("item.count"), key: "StackCount", width: 90 },
  {
    title: "", key: "_del", width: 80,
    render: (row) =>
      h(NButton, { size: "small", type: "error", secondary: true, onClick: () => handleDeleteItem(row) },
        { default: () => t("button.remove") }),
  },
];

onMounted(async () => {
  if (props.players.length > 0) {
    playerList.value = [...props.players];
  } else {
    const { data } = await new ApiService().getPlayerList({ order_by: "last_online", desc: true });
    playerList.value = Array.isArray(data.value) ? data.value : [];
  }
  localeLowerPalMap.value = Object.keys(palMap[locale.value] || {}).reduce((acc, key) => {
    acc[key.toLowerCase()] = palMap[locale.value][key];
    return acc;
  }, {});
});

watch(
  () => props.players,
  (v) => { if (v?.length > 0) playerList.value = [...v]; },
  { deep: true },
);
</script>

<template>
  <div class="item-manager h-full" :class="{ 'is-dark': isDarkMode }">
    <n-layout has-sider class="h-full">
      <!-- 左侧：玩家列表 -->
      <n-layout-sider
        :width="smallScreen ? 280 : 320"
        content-style="padding: 0 16px 16px;"
        :native-scrollbar="false"
        bordered
        class="relative"
      >
        <div class="im-search-wrap">
          <n-input
            v-model:value="searchPlayerValue"
            clearable size="large"
            :placeholder="$t('filter.searchPlayers')"
          />
        </div>
        <n-list :show-divider="false" class="mt-2">
          <n-list-item
            v-for="p in filteredPlayers" :key="p.player_uid"
            class="im-player-item" role="button" tabindex="0"
            @click="selectPlayer(p)" @keydown.enter="selectPlayer(p)"
          >
            <div class="im-player-row"
              :class="{ 'is-selected': selectedPlayer?.player_uid === p.player_uid }"
            >
              <div class="im-player-main">
                <span class="im-player-name">{{ p.nickname || "--" }}</span>
                <div class="im-player-meta">
                  <n-tag :bordered="false" type="primary" size="small" round>Lv.{{ p.level }}</n-tag>
                </div>
              </div>
              <n-icon class="im-chevron" size="16"><ChevronForward /></n-icon>
            </div>
          </n-list-item>
        </n-list>
      </n-layout-sider>

      <!-- 右侧：道具操作区 -->
      <n-layout :native-scrollbar="false" class="relative">
        <div v-if="!selectedPlayer" class="im-empty-hint">
          <n-empty :description="$t('message.selectPlayerFirst')" />
        </div>
        <div v-else class="im-content">
          <div class="im-player-header">
            <span class="im-player-title">{{ selectedPlayer.nickname }}</span>
            <n-tag type="primary" round>Lv.{{ selectedPlayer.level }}</n-tag>
            <n-button size="small" secondary :loading="syncing"
              style="margin-left:auto"
              title="重新解析存档文件，获取最新物品/帕鲁数据"
              @click="syncAndRefresh">
              🔄 同步存档数据
            </n-button>
          </div>

          <n-tabs v-model:value="activeTab" type="line" size="large" animated>

            <!-- 给予道具 -->
            <n-tab-pane name="give" :tab="$t('button.giveItem')">
              <div class="give-panel">
                <n-input v-model:value="itemSearchValue" clearable size="medium"
                  :placeholder="$t('input.selectItem')" class="mb-3" />
                <div class="item-grid">
                  <div v-for="item in filteredItems" :key="item.id"
                    class="item-card" :class="{ 'is-selected': selectedItem?.id === item.id }"
                    @click="selectedItem = item"
                  >
                    <n-avatar :src="getItemIcon(item.id)" :fallback-src="getUnknownAvatar()" size="small" />
                    <span class="item-name">{{ item.name }}</span>
                  </div>
                </div>
                <div class="give-actions">
                  <n-input-number v-model:value="giveAmount" :min="1" :max="9999" class="give-amount" />
                  <n-button type="primary" :loading="givingItem" :disabled="!selectedItem" round @click="handleGiveItem">
                    {{ $t("button.giveItem") }}
                  </n-button>
                </div>
                <div v-if="selectedItem" class="selected-item-info">
                  <n-tag type="info" round>{{ selectedItem.id }}</n-tag>
                  <span class="ml-2 text-sm opacity-60">{{ selectedItem.description }}</span>
                </div>
              </div>
            </n-tab-pane>

            <!-- 背包物品 -->
            <n-tab-pane name="inventory" :tab="$t('item.itemList')">
              <n-spin :show="loadingDetail">
                <n-tabs v-model:value="containerTab" type="segment" size="small" animated class="mt-2">
                  <n-tab-pane name="CommonContainerId" :tab="$t('item.commonContainer')">
                    <n-data-table size="small" :columns="createItemColumns()"
                      :data="mergedItems['CommonContainerId'] || []"
                      :bordered="false" striped :pagination="{ pageSize: 10 }" />
                  </n-tab-pane>
                  <n-tab-pane name="EssentialContainerId" :tab="$t('item.essentialContainer')">
                    <n-data-table size="small" :columns="createItemColumns()"
                      :data="mergedItems['EssentialContainerId'] || []"
                      :bordered="false" striped :pagination="{ pageSize: 10 }" />
                  </n-tab-pane>
                  <n-tab-pane name="WeaponLoadOutContainerId" :tab="$t('item.weaponContainer')">
                    <n-data-table size="small" :columns="createItemColumns()"
                      :data="mergedItems['WeaponLoadOutContainerId'] || []"
                      :bordered="false" striped />
                  </n-tab-pane>
                  <n-tab-pane name="PlayerEquipArmorContainerId" :tab="$t('item.armorContainer')">
                    <n-data-table size="small" :columns="createItemColumns()"
                      :data="mergedItems['PlayerEquipArmorContainerId'] || []"
                      :bordered="false" striped />
                  </n-tab-pane>
                </n-tabs>
              </n-spin>
            </n-tab-pane>

            <!-- 幻兽列表 -->
            <n-tab-pane name="pals" :tab="$t('item.palList')">
              <n-spin :show="loadingDetail">
                <n-input v-model:value="palSearchValue" clearable size="medium"
                  :placeholder="$t('input.searchPlaceholder')" class="my-2" />
                <n-empty v-if="filteredPals.length === 0" class="py-8" />
                <div v-else class="pals-list">
                  <div v-for="pal in filteredPals"
                    :key="pal.type + pal.level + pal.gender"
                    class="pal-card" :class="{ 'is-lucky': pal.is_lucky }"
                  >
                    <n-avatar :src="getPalAvatar(pal.type)" :fallback-src="getUnknownAvatar(pal.is_boss)" :size="48" />
                    <div class="pal-info">
                      <div class="pal-name">
                        {{ getPalName(pal.type) }}
                        <n-tag v-if="pal.is_lucky" type="warning" size="tiny" round :bordered="false">✨</n-tag>
                      </div>
                      <div class="pal-meta">
                        <n-tag :type="pal.gender === 'Male' ? 'primary' : 'error'" size="tiny" round :bordered="false">
                          {{ pal.gender === "Male" ? "♂" : "♀" }}
                        </n-tag>
                        <n-tag type="default" size="tiny" round :bordered="false">Lv.{{ pal.level }}</n-tag>
                      </div>
                      <div class="pal-skills">
                        <n-tag v-for="sk in pal.skills || []" :key="sk"
                          type="warning" size="tiny" round :bordered="false" class="mr-1">
                          {{ localizedSkillName(sk, locale, skillMap) }}
                        </n-tag>
                      </div>
                    </div>
                    <n-button size="small" type="error" secondary round :loading="releasingPal" @click="handleReleasePal(pal)">
                      {{ $t("button.releasePal") }}
                    </n-button>
                  </div>
                </div>
              </n-spin>
            </n-tab-pane>

          </n-tabs>
        </div>
      </n-layout>
    </n-layout>
  </div>
</template>

<style scoped lang="less">
.item-manager { background: transparent; }

.im-search-wrap {
  position: sticky; top: 0; z-index: 3;
  padding: 16px 0 8px;
  background: #fff;
}
.is-dark .im-search-wrap { background: #18181c; }

.im-player-item { padding: 0 0 4px !important; outline: none; }

.im-player-row {
  display: flex; align-items: center; gap: 8px;
  padding: 9px 12px;
  border: 1px solid transparent; border-radius: 10px;
  cursor: pointer;
  transition: background .15s, border-color .15s;
  &:hover { background: rgba(64,152,252,.08); }
  &.is-selected {
    border-color: rgba(64,152,252,.55);
    background: rgba(64,152,252,.12);
    box-shadow: inset 3px 0 0 #4098fc;
  }
}
.is-dark .im-player-row:hover { background: rgba(64,152,252,.12); }
.is-dark .im-player-row.is-selected { background: rgba(64,152,252,.18); }

.im-player-main { flex: 1; min-width: 0; }
.im-player-name {
  font-size: 15px; font-weight: 600;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.im-player-meta { display: flex; gap: 6px; margin-top: 4px; }
.im-chevron { flex: none; color: rgba(24,24,28,.28); }
.is-dark .im-chevron { color: rgba(255,255,255,.3); }

.im-empty-hint {
  display: flex; align-items: center; justify-content: center;
  height: 100%; min-height: 300px;
}

.im-content { padding: 20px 24px; }

.im-player-header {
  display: flex; align-items: center; gap: 12px;
  margin-bottom: 20px;
}
.im-player-title { font-size: 22px; font-weight: 700; }

.give-panel { padding-top: 12px; }

.item-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 8px; max-height: 320px; overflow-y: auto; padding: 4px 2px;
}

.item-card {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px;
  border: 1px solid rgba(24,24,28,.08); border-radius: 8px;
  cursor: pointer; transition: background .15s, border-color .15s;
  &:hover { background: rgba(64,152,252,.07); }
  &.is-selected { border-color: #4098fc; background: rgba(64,152,252,.12); }
}
.is-dark .item-card { border-color: rgba(255,255,255,.08); }
.item-name {
  font-size: 13px; flex: 1;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

.give-actions { display: flex; align-items: center; gap: 12px; margin-top: 14px; }
.give-amount { width: 120px; }
.selected-item-info { margin-top: 10px; }

.pals-list { display: flex; flex-direction: column; gap: 10px; }

.pal-card {
  display: flex; align-items: center; gap: 14px;
  padding: 12px 16px;
  border: 1px solid rgba(24,24,28,.07); border-radius: 12px;
  background: rgba(255,255,255,.5);
  transition: border-color .15s;
  &.is-lucky {
    border-color: rgba(230,162,60,.45);
    background: rgba(230,162,60,.06);
  }
}
.is-dark .pal-card {
  border-color: rgba(255,255,255,.07);
  background: rgba(255,255,255,.04);
}
.pal-info { flex: 1; min-width: 0; }
.pal-name { font-size: 15px; font-weight: 600; display: flex; align-items: center; gap: 6px; }
.pal-meta { display: flex; gap: 5px; margin-top: 5px; }
.pal-skills { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 5px; }
</style>
