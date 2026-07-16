<script setup>
import { ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { useMessage } from "naive-ui";
import ApiService from "@/service/api";
import techList from "@/assets/tech.json";

const { t } = useI18n();
const message = useMessage();

const props = defineProps({
  playerUid: { type: String, required: true },
});
const emit = defineEmits(["done"]);

const searchVal = ref("");
const selectedId = ref(null);
const loading = ref(false);

const labelOptions = computed(() => {
  const set = new Set(techList.map((e) => e.label).filter(Boolean));
  return [{ label: t("techPicker.allLabels"), value: "" }, ...[...set].map((l) => ({ label: l, value: l }))];
});
const filterLabel = ref("");

const filtered = computed(() => {
  const q = searchVal.value.trim().toLowerCase();
  return techList.filter((e) => {
    const matchLabel = !filterLabel.value || e.label === filterLabel.value;
    const matchQ =
      !q ||
      e.zh.toLowerCase().includes(q) ||
      e.id.toLowerCase().includes(q) ||
      String(e.level).includes(q);
    return matchLabel && matchQ;
  });
});

const selectedEntry = computed(() => techList.find((e) => e.id === selectedId.value) || null);

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
      message.success(t("techPicker.learnSuccess", { name: selectedEntry.value?.zh || selectedId.value, msg: body?.message || "OK" }));
      emit("done");
    } else {
      message.error(t("techPicker.learnFail", { err: body?.error || JSON.stringify(body) || "" }));
    }
  } catch (e) {
    message.error(t("techPicker.learnFail", { err: e.message }));
  } finally {
    loading.value = false;
  }
};

const columns = [
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
    key: "zh",
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
];

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
          <div style="font-weight:600;">{{ selectedEntry.zh }}</div>
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
