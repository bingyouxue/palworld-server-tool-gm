<script setup>
import { ref, computed } from "vue";
import { useMessage } from "naive-ui";
import ApiService from "@/service/api";
import gmPalsRaw from "@/assets/gm/pals.json";
import passivesRaw from "@/assets/gm/passives.json";
import activeSkillsRaw from "@/assets/gm/activeSkills.json";
import passiveI18nRaw from "@/assets/gm/passiveI18n.json";
import palNameMap from "@/assets/pal.json";
import { useI18n } from "vue-i18n";

const props = defineProps({
  show: { type: Boolean, default: false },
  playerUid: { type: String, required: true },
  onlinePlayers: { type: Array, default: () => [] },
});
const emit = defineEmits(["update:show", "done"]);

const { locale } = useI18n();
const message = useMessage();

function getPassiveRank(entry) {
  const r = Number(entry.rank);
  return Number.isFinite(r) ? r : 0;
}

const ELEMENT_ICON = {
  Fire:    "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_01.webp",
  Water:   "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_02.webp",
  Thunder: "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_03.webp",
  Leaf:    "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_04.webp",
  Dark:    "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_05.webp",
  Dragon:  "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_06.webp",
  Earth:   "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_07.webp",
  Ice:     "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_08.webp",
  Normal:  "https://cdn.paldb.cc/image/Pal/Texture/UI/InGame/T_Icon_element_s_00.webp",
};

// 工作适应性项目（对应 ExtraWorkSuitabilities 键名）
const WORK_SUITS = [
  { key: "EmitFlame",           label: "点火" },
  { key: "Watering",            label: "浇水" },
  { key: "Seeding",             label: "种植" },
  { key: "GenerateElectricity", label: "发电" },
  { key: "Handcraft",           label: "手工" },
  { key: "Collection",          label: "采集" },
  { key: "Deforest",            label: "伐木" },
  { key: "Mining",              label: "采矿" },
  { key: "OilExtraction",       label: "采油" },
  { key: "ProductMedicine",     label: "制药" },
  { key: "Cool",                label: "冷却" },
  { key: "Transport",           label: "搬运" },
  { key: "MonsterFarm",         label: "牧场" },
  { key: "Anyone",              label: "通用" },
];

const palZhMap = computed(() => palNameMap[locale.value] || palNameMap["zh"] || {});

const allPals = computed(() =>
  gmPalsRaw.map((p) => ({
    ...p,
    label:   palZhMap.value[p.id] || p.zh || p.name || p.id,
    labelEn: p.name || p.id,
  })),
);

const palIconMap = Object.fromEntries(
  gmPalsRaw.filter((p) => p.icon).map((p) => [p.id, `/gm-data/pals/${p.icon}`]),
);
const getPalAvatar = (id) => palIconMap[id] || "/gm-data/pals/T_icon_unknown.webp";

const passiveLocaleMap = computed(() => {
  const lang = locale.value || "zh";
  return passiveI18nRaw[lang] || passiveI18nRaw["zh"] || passiveI18nRaw["en"] || {};
});

const allPassives = computed(() =>
  passivesRaw.map((p) => {
    const i18n = passiveLocaleMap.value[p.id] || {};
    const enI18n = (passiveI18nRaw["en"] || {})[p.id] || {};
    return { ...p, label: i18n.name || enI18n.name || p.name || p.id, desc: i18n.desc || enI18n.desc || "" };
  }),
);

const allActiveSkills = computed(() =>
  activeSkillsRaw.map((s) => ({ ...s, label: s.zh || s.name || s.id })),
);

const CATEGORIES = [
  { key: "pal",      label: "普通" },
  { key: "boss",     label: "头目" },
  { key: "predator", label: "狂暴化" },
  { key: "gym",      label: "高塔" },
  { key: "raid",     label: "突袭" },
  { key: "human",    label: "Human" },
];

// ── 帕鲁选择 ──────────────────────────────────────────────
const palSearch      = ref("");
const activeCategory = ref("pal");
const selectedPal    = ref(null);
const filteredPals   = computed(() => {
  const kw = palSearch.value.trim().toLowerCase();
  let list = allPals.value;
  if (kw) {
    list = list.filter((p) =>
      (p.label || "").toLowerCase().includes(kw) || (p.id || "").toLowerCase().includes(kw),
    );
  } else {
    list = list.filter((p) => (p.type || "pal") === activeCategory.value);
  }
  return list.slice(0, 300);
});

// ── 被动技能（最多 8 个）─────────────────────────────────
const passiveSearch  = ref("");
const passiveDropOpen = ref(false);
const selectedPassives = ref([]);
const filteredPassives = computed(() => {
  const kw = passiveSearch.value.trim().toLowerCase();
  const ids = new Set(selectedPassives.value.map((p) => p.id));
  let list = allPassives.value.filter((p) => !ids.has(p.id));
  if (kw) list = list.filter((p) =>
    (p.label || "").toLowerCase().includes(kw) ||
    (p.id || "").toLowerCase().includes(kw) ||
    (p.desc || "").toLowerCase().includes(kw),
  );
  return list.slice(0, 80);
});
const addPassive = (p) => {
  if (selectedPassives.value.length >= 8) { message.warning("最多选择 8 个被动技能"); return; }
  if (!selectedPassives.value.find((x) => x.id === p.id)) selectedPassives.value = [...selectedPassives.value, p];
  passiveSearch.value = "";
  passiveDropOpen.value = false;
};
const removePassive = (id) => { selectedPassives.value = selectedPassives.value.filter((p) => p.id !== id); };
const onPassiveClickOutside = () => { passiveDropOpen.value = false; };

// ── 主动技能（最多 3 个）─────────────────────────────────
const skillSearch  = ref("");
const skillDropOpen = ref(false);
const selectedSkills = ref([]);
const filteredSkills = computed(() => {
  const kw = skillSearch.value.trim().toLowerCase();
  const ids = new Set(selectedSkills.value.map((s) => s.id));
  let list = allActiveSkills.value.filter((s) => !ids.has(s.id));
  if (kw) list = list.filter((s) =>
    (s.label || "").toLowerCase().includes(kw) ||
    (s.name || "").toLowerCase().includes(kw) ||
    (s.id || "").toLowerCase().includes(kw) ||
    (s.element || "").toLowerCase().includes(kw),
  );
  return list.slice(0, 80);
});
const addSkill = (s) => {
  if (selectedSkills.value.length >= 3) { message.warning("最多选择 3 个主动技能"); return; }
  if (!selectedSkills.value.find((x) => x.id === s.id)) selectedSkills.value = [...selectedSkills.value, s];
  skillSearch.value = "";
  skillDropOpen.value = false;
};
const removeSkill = (id) => { selectedSkills.value = selectedSkills.value.filter((s) => s.id !== id); };
const onSkillClickOutside = () => { skillDropOpen.value = false; };

// ── 基础属性 ──────────────────────────────────────────────
const nickname          = ref("");
const gender            = ref("Male");
const level             = ref(1);
const isLucky           = ref(false);
const isAwakening       = ref(false);
const partnerSkillLevel = ref(null);

// ── IV 值（0-255）────────────────────────────────────────
const ivHealth   = ref(null);
const ivAtkMelee = ref(null);
const ivAtkShot  = ref(null);
const ivDef      = ref(null);

// ── 星级 / 灵魂强化（0-255）──────────────────────────────
const stars       = ref(null);
const soulHealth  = ref(null);
const soulAttack  = ref(null);
const soulDefense = ref(null);
const soulCraft   = ref(null);

// ── 工作适应性（0-5，对应 ExtraWorkSuitabilities）────────
const workSuits = ref(Object.fromEntries(WORK_SUITS.map((w) => [w.key, null])));

const submitting = ref(false);
const close = () => emit("update:show", false);

const currentUserId = computed(() => {
  const uid = String(props.playerUid);
  const online = props.onlinePlayers.find((p) => {
    const puid = String(p.player_uid || p.PlayerUid || "");
    return puid === uid || puid === uid.replace(/^0+/, "");
  });
  return online?.user_id || online?.UserId || "";
});

const hasAdvancedAttrs = computed(() =>
  selectedPassives.value.length > 0 ||
  selectedSkills.value.length > 0 ||
  ivHealth.value != null ||
  ivAtkMelee.value != null ||
  ivAtkShot.value != null ||
  ivDef.value != null ||
  stars.value != null ||
  soulHealth.value != null ||
  soulAttack.value != null ||
  soulDefense.value != null ||
  soulCraft.value != null ||
  isAwakening.value ||
  partnerSkillLevel.value != null ||
  Object.values(workSuits.value).some((v) => v != null && v > 0),
);

const handleSubmit = async () => {
  if (!selectedPal.value) { message.warning("请先选择帕鲁种类"); return; }
  const userId = currentUserId.value;
  if (!userId) { message.warning("该玩家当前不在线，无法给予帕鲁"); return; }
  submitting.value = true;
  try {
    if (hasAdvancedAttrs.value) {
      // 高级属性：调后端写 PalDefender 模板文件，发 givepal_j
      const extraWork = {};
      for (const w of WORK_SUITS) {
        const v = workSuits.value[w.key];
        if (v != null && v > 0) extraWork[w.key] = v;
      }
      const res = await new ApiService().giveCustomPal({
        playerUid:           props.playerUid,
        pal_id:              selectedPal.value.id,
        nickname:            nickname.value || undefined,
        gender:              gender.value || undefined,
        level:               level.value || 1,
        is_awakening:        isAwakening.value || undefined,
        partner_skill_level: partnerSkillLevel.value ?? undefined,
        passives:            selectedPassives.value.map((p) => p.id),
        active_skills:       selectedSkills.value.map((s) => s.id),
        stars:               stars.value ?? undefined,
        ivs: {
          health:       ivHealth.value ?? undefined,
          attack_melee: ivAtkMelee.value ?? undefined,
          attack_shot:  ivAtkShot.value ?? undefined,
          defense:      ivDef.value ?? undefined,
        },
        souls: {
          health:      soulHealth.value ?? undefined,
          attack:      soulAttack.value ?? undefined,
          defense:     soulDefense.value ?? undefined,
          craft_speed: soulCraft.value ?? undefined,
        },
        extra_work_suitabilities: Object.keys(extraWork).length > 0 ? extraWork : undefined,
      });
      const code = res.statusCode?.value ?? res.statusCode;
      const body = res.data?.value ?? res.data;
      if (code === 200) {
        message.success(`已给予自定义帕鲁: ${selectedPal.value.label}  ${body?.message || ""}`);
        emit("done"); close();
      } else {
        message.error("给予失败: " + (body?.error || body?.message || "需要安装 PalDefender 并配置正确的存档路径"));
      }
    } else {
      // 基础给予：直接 RCON givepal
      const cmd = `givepal ${userId} ${selectedPal.value.id} ${level.value || 1}`;
      const res = await new ApiService().sendRconCommand({ command: cmd });
      const code = res.statusCode?.value ?? res.statusCode;
      const body = res.data?.value ?? res.data;
      if (code === 200) {
        message.success(`已给予帕鲁: ${selectedPal.value.label}  ${body?.message || ""}`);
        emit("done"); close();
      } else {
        message.error("给予失败: " + (body?.error || body?.message || "需要安装 PalDefender 插件"));
      }
    }
  } catch (e) {
    message.error("给予失败: " + e.message);
  } finally {
    submitting.value = false;
  }
};
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    style="width: 95%; max-width: 720px"
    title="给予自定义帕鲁"
    header-style="padding: 14px 20px;"
    content-style="padding: 16px 20px; max-height: 80vh; overflow-y: auto;"
    footer-style="padding: 12px 20px;"
    :bordered="false"
    @update:show="emit('update:show', $event)"
  >
    <div class="gm-cpal">
      <n-alert type="info" :show-icon="true" class="mb-3" size="small">
        被动/主动技能、IV、灵魂强化、觉醒、工作适应性等属性需要服务器安装 PalDefender 插件才能生效，否则仅发送基础 givepal 命令。
      </n-alert>

      <!-- 帕鲁选择 -->
      <n-card size="small" class="mb-3" title="选择帕鲁">
        <div class="cpal-cat-tabs mb-2">
          <div
            v-for="cat in CATEGORIES" :key="cat.key"
            class="cpal-cat-tab"
            :class="{ active: activeCategory === cat.key && !palSearch }"
            @click="activeCategory = cat.key; palSearch = ''"
          >{{ cat.label }}</div>
        </div>
        <n-input v-model:value="palSearch" clearable placeholder="搜索帕鲁名称 / ID（搜索时忽略分类）" size="small" class="mb-2" />
        <div class="cpal-pal-grid">
          <div
            v-for="pal in filteredPals" :key="pal.id"
            class="cpal-pal-card"
            :class="{ 'is-selected': selectedPal?.id === pal.id }"
            @click="selectedPal = pal"
          >
            <img :src="getPalAvatar(pal.id)" :alt="pal.label" class="cpal-pal-img"
              @error="$event.target.style.display='none'" />
            <span class="cpal-pal-name">{{ pal.label }}</span>
            <span class="cpal-pal-en">{{ pal.labelEn }}</span>
          </div>
        </div>
        <div v-if="selectedPal" class="mt-2">
          <n-tag type="primary" round size="small">已选: {{ selectedPal.label }} ({{ selectedPal.id }})</n-tag>
        </div>
      </n-card>

      <!-- 基础属性 -->
      <n-card size="small" class="mb-3" title="基础属性">
        <n-grid :cols="2" :x-gap="12" :y-gap="6">
          <n-gi>
            <n-form-item label="昵称" label-placement="left" label-width="80" size="small">
              <n-input v-model:value="nickname" placeholder="可留空" clearable size="small" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="性别" label-placement="left" label-width="80" size="small">
              <n-select v-model:value="gender" size="small"
                :options="[
                  { label: '♂ 雄', value: 'Male' },
                  { label: '♀ 雌', value: 'Female' },
                  { label: '无性别', value: 'None' }
                ]" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="等级" label-placement="left" label-width="80" size="small">
              <n-input-number v-model:value="level" :min="1" :max="255" size="small" style="width:100%" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="伙伴技能Lv" label-placement="left" label-width="80" size="small">
              <n-input-number v-model:value="partnerSkillLevel" :min="1" :max="255" placeholder="可留空"
                size="small" style="width:100%" clearable />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="幸运" label-placement="left" label-width="80" size="small">
              <n-switch v-model:value="isLucky" size="small" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item label="觉醒" label-placement="left" label-width="80" size="small">
              <n-switch v-model:value="isAwakening" size="small" />
            </n-form-item>
          </n-gi>
        </n-grid>
      </n-card>

      <!-- 被动技能（最多 8 个）-->
      <n-card size="small" class="mb-3">
        <template #header>
          <span>被动技能</span>
          <n-tag size="tiny" :type="selectedPassives.length >= 8 ? 'error' : 'default'" class="ml-2">{{ selectedPassives.length }}/8</n-tag>
        </template>
        <div class="cpal-chips mb-2">
          <div v-for="p in selectedPassives" :key="p.id" class="cpal-chip" :class="`rank-${getPassiveRank(p)}`">
            <span>{{ p.label }}</span>
            <n-button text size="tiny" type="error" @click="removePassive(p.id)">×</n-button>
          </div>
          <div v-if="selectedPassives.length === 0" class="cpal-chips-empty">未选择被动技能</div>
        </div>
        <div class="cpal-picker-wrap" v-click-outside="onPassiveClickOutside">
          <n-input v-model:value="passiveSearch" clearable placeholder="搜索被动技能（点击添加）" size="small"
            @focus="passiveDropOpen = true" @input="passiveDropOpen = true" />
          <div v-if="passiveDropOpen" class="cpal-dropdown">
            <div v-for="p in filteredPassives" :key="p.id" class="cpal-drop-item" :class="`rank-bg-${getPassiveRank(p)}`"
              @mousedown.prevent="addPassive(p)">
              <div class="cpal-di-row">
                <span class="cpal-di-name">{{ p.label }}</span>
                <span v-if="getPassiveRank(p) !== 0" class="cpal-rank-badge" :class="`rank-badge-${getPassiveRank(p)}`">
                  Rank {{ getPassiveRank(p) > 0 ? '+' : '' }}{{ getPassiveRank(p) }}
                </span>
              </div>
              <div v-if="p.desc" class="cpal-di-desc">{{ p.desc }}</div>
              <div class="cpal-di-id">{{ p.id }}</div>
            </div>
            <div v-if="filteredPassives.length === 0" class="cpal-drop-empty">无匹配结果</div>
          </div>
        </div>
      </n-card>

      <!-- 主动技能（最多 3 个）-->
      <n-card size="small" class="mb-3">
        <template #header>
          <span>主动技能</span>
          <n-tag size="tiny" :type="selectedSkills.length >= 3 ? 'error' : 'default'" class="ml-2">{{ selectedSkills.length }}/3</n-tag>
        </template>
        <div class="cpal-chips mb-2">
          <div v-for="s in selectedSkills" :key="s.id" class="cpal-chip skill-chip">
            <img v-if="ELEMENT_ICON[s.element]" :src="ELEMENT_ICON[s.element]" class="cpal-skill-icon" />
            <span>{{ s.label }}</span>
            <n-button text size="tiny" type="error" @click="removeSkill(s.id)">×</n-button>
          </div>
          <div v-if="selectedSkills.length === 0" class="cpal-chips-empty">未选择主动技能</div>
        </div>
        <div class="cpal-picker-wrap" v-click-outside="onSkillClickOutside">
          <n-input v-model:value="skillSearch" clearable placeholder="搜索主动技能（点击添加）" size="small"
            @focus="skillDropOpen = true" @input="skillDropOpen = true" />
          <div v-if="skillDropOpen" class="cpal-dropdown">
            <div v-for="s in filteredSkills" :key="s.id" class="cpal-drop-item"
              @mousedown.prevent="addSkill(s)">
              <div class="cpal-di-head">
                <img v-if="ELEMENT_ICON[s.element]" :src="ELEMENT_ICON[s.element]" class="cpal-skill-icon" />
                <span class="cpal-di-name">{{ s.label }}</span>
                <n-tag v-if="s.element" size="tiny" :bordered="false" style="margin-left:4px">{{ s.element }}</n-tag>
              </div>
              <div class="cpal-di-id">{{ s.id }}</div>
            </div>
            <div v-if="filteredSkills.length === 0" class="cpal-drop-empty">无匹配结果</div>
          </div>
        </div>
      </n-card>

      <!-- IV 值（0-255）-->
      <n-card size="small" class="mb-3" title="IV 值（0-255，可选）">
        <n-grid :cols="2" :x-gap="12" :y-gap="6">
          <n-gi><n-form-item label="HP" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="ivHealth" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="近战攻击" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="ivAtkMelee" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="射击攻击" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="ivAtkShot" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="防御" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="ivDef" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
        </n-grid>
      </n-card>

      <!-- 星级 + 灵魂强化（0-255）-->
      <n-card size="small" class="mb-3" title="星级 / 灵魂强化（可选）">
        <n-grid :cols="2" :x-gap="12" :y-gap="6">
          <n-gi><n-form-item label="星级" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="stars" :min="0" :max="10" placeholder="0-10" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="HP 强化" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="soulHealth" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="攻击强化" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="soulAttack" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="防御强化" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="soulDefense" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
          <n-gi><n-form-item label="制作强化" label-placement="left" label-width="80" size="small">
            <n-input-number v-model:value="soulCraft" :min="0" :max="255" placeholder="0-255" size="small" style="width:100%" clearable />
          </n-form-item></n-gi>
        </n-grid>
      </n-card>

      <!-- 工作适应性（ExtraWorkSuitabilities，0-5）-->
      <n-card size="small" title="工作适应性（0-5，可选）">
        <n-grid :cols="3" :x-gap="12" :y-gap="6">
          <n-gi v-for="w in WORK_SUITS" :key="w.key">
            <n-form-item :label="w.label" label-placement="left" label-width="52" size="small">
              <n-input-number v-model:value="workSuits[w.key]" :min="0" :max="5" placeholder="0-5"
                size="small" style="width:100%" clearable />
            </n-form-item>
          </n-gi>
        </n-grid>
      </n-card>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <n-button size="small" @click="close">取消</n-button>
        <n-button type="primary" size="small" round :loading="submitting" :disabled="!selectedPal" @click="handleSubmit">
          给予自定义帕鲁
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped lang="less">
.gm-cpal { display: flex; flex-direction: column; }

.cpal-cat-tabs { display: flex; flex-wrap: wrap; gap: 4px; }
.cpal-cat-tab {
  padding: 3px 12px; border-radius: 20px; font-size: 12px;
  cursor: pointer; user-select: none;
  border: 1px solid rgba(64,152,252,.25); background: transparent;
  transition: all .15s;
  &:hover { border-color: #4098fc; background: rgba(64,152,252,.08); }
  &.active { border-color: #4098fc; background: #4098fc; color: #fff; }
}

.cpal-pal-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
  gap: 5px; max-height: 180px; overflow-y: auto;
}
.cpal-pal-card {
  display: flex; flex-direction: column; align-items: center; gap: 2px;
  padding: 5px 4px; border: 1px solid rgba(24,24,28,.1); border-radius: 6px;
  cursor: pointer; text-align: center; transition: border-color .15s, background .15s;
  &:hover { border-color: #4098fc; background: rgba(64,152,252,.06); }
  &.is-selected { border-color: #4098fc; background: rgba(64,152,252,.14); box-shadow: 0 0 0 2px rgba(64,152,252,.3); }
}
.is-dark .cpal-pal-card { border-color: rgba(255,255,255,.1); }
.cpal-pal-img { width: 40px; height: 40px; object-fit: contain; }
.cpal-pal-name {
  font-size: 11px; font-weight: 600; line-height: 1.2;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
  overflow: hidden; width: 100%;
}
.cpal-pal-en {
  font-size: 9px; opacity: 0.42; line-height: 1.2;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap; width: 100%;
}

.cpal-chips { display: flex; flex-wrap: wrap; gap: 6px; min-height: 28px; }
.cpal-chips-empty { font-size: 12px; opacity: 0.45; align-self: center; }
.cpal-chip {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 3px 8px 3px 10px; border-radius: 20px; font-size: 12px;
  border: 1px solid rgba(64,152,252,.3); background: rgba(64,152,252,.08);
}
.cpal-chip.rank-5  { border-color: rgba(208,48,80,.5);   background: rgba(208,48,80,.1); }
.cpal-chip.rank-4  { border-color: rgba(240,160,32,.5);  background: rgba(240,160,32,.1); }
.cpal-chip.rank-3  { border-color: rgba(64,152,252,.5);  background: rgba(64,152,252,.1); }
.cpal-chip.rank-2  { border-color: rgba(24,160,88,.5);   background: rgba(24,160,88,.1); }
.cpal-chip.rank-1  { border-color: rgba(100,100,100,.3); background: rgba(100,100,100,.06); }
.cpal-chip.rank--1 { border-color: rgba(160,160,160,.3); background: rgba(160,160,160,.06); }
.cpal-chip.rank--2 { border-color: rgba(208,48,80,.3);   background: rgba(208,48,80,.06); }
.cpal-chip.rank--3 { border-color: rgba(208,48,80,.4);   background: rgba(208,48,80,.08); }
.cpal-chip.skill-chip { gap: 5px; }
.cpal-skill-icon { width: 16px; height: 16px; object-fit: contain; }

.cpal-rank-badge {
  display: inline-flex; align-items: center;
  padding: 1px 6px; border-radius: 10px; font-size: 10px; font-weight: 700; border: 1px solid;
}
.cpal-rank-badge.rank-badge-5  { border-color: rgba(208,48,80,.5);  background: rgba(208,48,80,.12);  color: #d03050; }
.cpal-rank-badge.rank-badge-4  { border-color: rgba(240,160,32,.5); background: rgba(240,160,32,.12); color: #c07818; }
.cpal-rank-badge.rank-badge-3  { border-color: rgba(64,152,252,.5); background: rgba(64,152,252,.12); color: #2080c8; }
.cpal-rank-badge.rank-badge-2  { border-color: rgba(24,160,88,.5);  background: rgba(24,160,88,.12);  color: #18a058; }
.cpal-rank-badge.rank-badge-1  { border-color: rgba(100,100,100,.3);background: rgba(100,100,100,.08);color: #888; }
.cpal-rank-badge.rank-badge--1 { border-color: rgba(150,150,150,.3);background: rgba(150,150,150,.08);color: #999; }
.cpal-rank-badge.rank-badge--2 { border-color: rgba(208,48,80,.35); background: rgba(208,48,80,.07);  color: #c04060; }
.cpal-rank-badge.rank-badge--3 { border-color: rgba(208,48,80,.45); background: rgba(208,48,80,.09);  color: #d03050; }

.rank-bg-5  { background: rgba(208,48,80,.04); }
.rank-bg-4  { background: rgba(240,160,32,.04); }
.rank-bg-3  { background: rgba(64,152,252,.04); }
.rank-bg-2  { background: rgba(24,160,88,.04); }
.rank-bg--1 { background: rgba(150,150,150,.03); }
.rank-bg--2 { background: rgba(208,48,80,.03); }
.rank-bg--3 { background: rgba(208,48,80,.04); }

.cpal-picker-wrap { position: relative; }
.cpal-dropdown {
  position: absolute; left: 0; right: 0; top: calc(100% + 4px);
  max-height: 220px; overflow-y: auto; z-index: 2000;
  border: 1px solid rgba(24,24,28,.12); border-radius: 8px;
  background: var(--n-color, #fff); box-shadow: 0 4px 20px rgba(0,0,0,.15);
}
.is-dark .cpal-dropdown { border-color: rgba(255,255,255,.12); background: #27272a; }
.cpal-drop-item {
  padding: 7px 12px; cursor: pointer; transition: background .12s;
  &:hover { background: rgba(64,152,252,.15); }
}
.cpal-drop-empty { padding: 10px 12px; font-size: 12px; opacity: 0.5; }
.cpal-di-name { font-size: 12px; font-weight: 600; }
.cpal-di-id { font-size: 10px; opacity: 0.42; margin-top: 2px; }
.cpal-di-desc { font-size: 11px; opacity: 0.65; margin-top: 2px; line-height: 1.35; }
.cpal-di-head { display: flex; align-items: center; gap: 5px; flex-wrap: wrap; }
.cpal-di-row { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
</style>
