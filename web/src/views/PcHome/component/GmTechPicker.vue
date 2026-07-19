<script setup>
import { ref, computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { useMessage } from "naive-ui";
import ApiService from "@/service/api";
import techList from "@/assets/tech.json";
import techI18n from "@/assets/gm/techI18n.json";

const { t, locale } = useI18n();
const message = useMessage();

const props = defineProps({
  playerUid: { type: String, required: true },
});
const emit = defineEmits(["done"]);

const searchVal = ref("");
const selectedId = ref(null);
const loading = ref(false);
const filterLabel = ref("");

/**
 * 根据当前语言从 techI18n.json 取名称，
 * 回退顺序：当前语言 → zh → id
 */
function getTechName(id) {
  const entry = techI18n[id];
  if (!entry) return id;
  const lang = locale.value;
  return entry[lang] || entry.zh || id;
}

/**
 * 分类选项，label 字段直接来自 techI18n（已是中文），
 * 切换语言后分类名暂不翻译（op.gg 抓到的 label 就是中文原值）
 */
const labelOptions = computed(() => {
  const set = new Set(techList.map((e) => e.label).filter(Boolean));
  return [
    { label: t("techPicker.allLabels"), value: "" },
    ...[...set].map((l) => ({ label: l, value: l })),
  ];
});

/** 扁平化列表，注入本地化 name 供搜索和排序 */
const enrichedList = computed(() =>
  techList.map((e) => ({
    ...e,
    _name: getTechName(e.id),
  }))
);

const filtered = computed(() => {
  const q = searchVal.value.trim().toLowerCase();
  return enrichedList.value.filter((e) => {
    const matchLabel = !filterLabel.value || e.label === filterLabel.value;
    const matchQ =
      !q ||
      e._name.toLowerCase().includes(q) ||
      (e.zh || "").toLowerCase().includes(q) ||
      e.id.toLowerCase().includes(q) ||
      String(e.level).includes(q);
    return matchLabel && matchQ;
  });
});

const selectedEntry = computed(
  () => enrichedList.value.find((e) => e.id === selectedId.value) || null
);

const parseRes = (res) => ({
  code: res.statusCode?.value ?? res.statusCode,
  body: res.data?.value ?? res.data,
});

const doLearn = async () => {
  if (!selectedId.value) {
    message.warning(t("techPicker.selectFirst"));
    return;
  }
  loading.value = true;
  try {
    const res = await new ApiService().learnTech({
      playerUid: props.playerUid,
      tech_id: selectedId.value,
    });
    const { code, body } = parseRes(res);
    if (code === 200) {
      message.success(
        t("techPicker.learnSuccess", {
          name: selectedEntry.value?._name || selectedId.value,
          msg: body?.message || "OK",
        })
      );
    } else {
      message.error(
        t("techPicker.learnFail", { err: body?.error || JSON.stringify(body) || "" })
      );
    }
  } catch (e) {
    message.error(t("techPicker.learnFail", { err: e.message }));
  } finally {
    loading.value = false;
  }
};

const columns = computed(() => [
  {
    title: "",
    key: "icon",
    width: 44,
    render(row) {
      return h("img", {
        src: row.iconUrl,
        style: "width:28px;height:28px;object-fit:contain;vertical-align:middle;",
        onerror: (e) => { e.target.style.display = "none"; },
      });
    },
  },
  {
    title: t("techPicker.colName"),
    key: "_name",
    ellipsis: { tooltip: true },
  },
  {
    title: t("techPicker.colLevel"),
    key: "level",
    width: 64,
    sorter: "default",
    defaultSortOrder: "ascend",
  },
  {
    title: t("techPicker.colLabel"),
    key: "label",
    width: 72,
  },
  {
    title: "ID",
    key: "id",
    width: 100,
    ellipsis: { tooltip: true },
    render(row) {
      return h("span", { style: "font-size:11px;font-family:monospace;opacity:.7;" }, row.id);
    },
  },
]);

const rowProps = (row) => ({
  style: "cursor:pointer;",
  onClick: () => { selectedId.value = row.id; },
});

const pagination = { pageSize: 10 };
</script>

<template>
  <div class="tech-picker">
    <!-- 搜索 + 分类筛选 -->
    <n-space align="center" style="margin-bottom:12px;" wrap>
      <n-input
        v-model:value="searchVal"
        clearable
        :placeholder="t('techPicker.searchPlaceholder')"
        style="width:220px"
      />
      <n-select
        v-model:value="filterLabel"
        :options="labelOptions"
        style="width:120px"
      />
    </n-space>

    <!-- 科技列表 -->
    <n-data-table
      :columns="columns"
      :data="filtered"
      :pagination="pagination"
      :bordered="false"
      size="small"
      striped
      :row-props="rowProps"
      :row-class-name="(row) => row.id === selectedId ? 'tech-row--selected' : ''"
    />

    <!-- 已选中预览 + 确认按钮 -->
    <div v-if="selectedEntry" class="tech-selected-preview">
      <n-space align="center">
        <img
          :src="selectedEntry.iconUrl"
          style="width:36px;height:36px;object-fit:contain;"
        />
        <div>
          <div style="font-weight:600;">{{ selectedEntry._name }}</div>
          <div style="font-size:12px;opacity:.6;">Lv.{{ selectedEntry.level }} · {{ selectedEntry.label }} · {{ selectedEntry.id }}</div>
        </div>
      </n-space>
      <n-button type="primary" :loading="loading" @click="doLearn" style="margin-top:12px;width:100%;">
        {{ t("techPicker.confirmLearn") }}
      </n-button>
    </div>
    <div v-else style="margin-top:12px;text-align:center;opacity:.5;font-size:13px;">
      {{ t("techPicker.clickToSelect") }}
    </div>
  </div>
</template>

<style scoped>
.tech-picker {
  min-height: 300px;
}

.tech-selected-preview {
  margin-top: 14px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid rgba(64, 152, 252, 0.3);
  background: rgba(64, 152, 252, 0.06);
}

:deep(.tech-row--selected td) {
  background: rgba(64, 152, 252, 0.12) !important;
}
</style>
