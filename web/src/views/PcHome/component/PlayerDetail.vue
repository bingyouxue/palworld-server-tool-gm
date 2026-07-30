<script setup>
import { computed, h, inject, nextTick, onMounted, reactive, ref, watch } from "vue";
import { ContentCopyFilled, PersonSearchSharp } from "@vicons/material";
import { LogOut, Ban, ShieldCheckmarkOutline } from "@vicons/ionicons5";
import { CrownFilled } from "@vicons/antd";
import ApiService from "@/service/api";
import dayjs from "dayjs";
import { useI18n } from "vue-i18n";
import palMap from "@/assets/pal.json";
import skillMap from "@/assets/skill.json";
import activeSkills from "@/assets/gm/activeSkills.json";
import passiveSkills from "@/assets/gm/passives.json";
import passiveI18n from "@/assets/gm/passiveI18n.json";
import GmItemPicker from "./GmItemPicker.vue";
import GmPalPicker from "./GmPalPicker.vue";
import GmTeleportModal from "./GmTeleportModal.vue";
import GmCustomPalModal from "./GmCustomPalModal.vue";
import GmTechPicker from "./GmTechPicker.vue";
import { useDialog, useMessage, NAvatar, NTag, NButton, NSpace, NInputNumber, NTooltip } from "naive-ui";
import PalDetail from "./PalDetail.vue";
import whitelistStore from "@/stores/model/whitelist.js";
import playerToGuildStore from "@/stores/model/playerToGuild.js";
import userStore from "@/stores/model/user";
import palItems from "@/assets/items.json";
import {
  localizedSkillName,
  statusPointTranslationKey,
} from "@/utils/gameLabels";

const { t, locale } = useI18n();
const PALWORLD_TOKEN = "palworld_token";
const props = defineProps({
  playerInfo: Object,
  playerPalsList: Array,
  isOnline: Boolean,
});
const emit = defineEmits(["refresh-pals"]);
const playerInfo = computed(() => props.playerInfo);
const playerPalsList = computed(() => props.playerPalsList);

const isLogin = computed(() => userStore().getLoginInfo().isLogin);

const message = useMessage();
const dialog = useDialog();

const isDarkMode = inject("isDarkMode", ref(
  window.matchMedia("(prefers-color-scheme: dark)").matches,
));

const platformColors = {
  steam: { color: "#223D58", textColor: "#fff" },
  xbox: { color: "#2B8B2B", textColor: "#fff" },
  ps5: { color: "#00439C", textColor: "#fff" },
  mac: { color: "#777", textColor: "#fff" },
  default: { color: "#d9c36c", textColor: "#fff" },
};

const localeLowerPalMap = ref({});
const skillTypeList = ref([]);

// GM 操作下拉菜单
const activeTab = ref(null);
const showGiveItemModal = ref(false);
const showGivePalModal = ref(false);
const showTpModal = ref(false);
const showCustomPalModal = ref(false);
const onlinePlayers = ref([]);

const gmOpsOptions = computed(() => [
  { label: t("button.giveItem"), key: "give_item" },
  { label: t("button.givePal"), key: "give_pal" },
  { label: t("button.teleportPlayer"), key: "tp" },
  { label: t("button.giveCustomPal"), key: "give_custom_pal" },
  { type: "divider", key: "d1" },
  { label: t("button.giveExp"), key: "give_exp" },
  { label: t("button.giveTech"), key: "give_tech" },
  { label: t("button.giveAncientTech"), key: "give_ancient_tech" },
  { label: t("button.learnTech"), key: "learn_tech" },
  { type: "divider", key: "d2" },
  { label: t("item.palList"), key: "view_pals" },
  { label: t("item.itemList"), key: "view_items" },
]);

// 给予经验/科技点 modal
const showGiveExpModal = ref(false);
const showGiveTechModal = ref(false);
const showGiveAncientTechModal = ref(false);
const showLearnTechModal = ref(false);
const giveExpAmount = ref(1000);
const giveTechAmount = ref(10);
const giveAncientTechAmount = ref(5);

const doGiveExp = async () => {
  const cmd = `giveexp ${playerInfo.value.player_uid} ${giveExpAmount.value}`;
  console.log("[GM] doGiveExp cmd:", cmd, "payload:", { exp: giveExpAmount.value });
  try {
    const res = await new ApiService().giveExp({ playerUid: playerInfo.value.player_uid, exp: giveExpAmount.value });
    const code = res.statusCode?.value ?? res.statusCode;
    const body = res.data?.value ?? res.data;
    console.log("[GM] doGiveExp response:", code, body);
    if (code === 200) {
      message.success(t("message.giveExpSuccess", { msg: body?.message || "OK" }));
    } else {
      message.error(t("message.giveExpFail", { err: body?.error || JSON.stringify(body) || "" }));
    }
  } catch (e) {
    console.error("[GM] doGiveExp error:", e);
    message.error("给予失败: " + e.message);
  }
};

const doGiveTech = async () => {
  try {
    const res = await new ApiService().giveTechPoint({ playerUid: playerInfo.value.player_uid, point: giveTechAmount.value });
    const code = res.statusCode?.value ?? res.statusCode;
    const body = res.data?.value ?? res.data;
    console.log("[GM] doGiveTech response:", code, body);
    if (code === 200) {
      message.success(t("message.giveTechSuccess", { msg: body?.message || "OK" }));
    } else {
      message.error(t("message.giveExpFail", { err: body?.error || JSON.stringify(body) || "" }));
    }
  } catch (e) {
    console.error("[GM] doGiveTech error:", e);
    message.error("给予失败: " + e.message);
  }
};

const doGiveAncientTech = async () => {
  try {
    const res = await new ApiService().giveAncientTechPoint({ playerUid: playerInfo.value.player_uid, point: giveAncientTechAmount.value });
    const code = res.statusCode?.value ?? res.statusCode;
    const body = res.data?.value ?? res.data;
    console.log("[GM] doGiveAncientTech response:", code, body);
    if (code === 200) {
      message.success(t("message.giveAncientTechSuccess", { msg: body?.message || "OK" }));
    } else {
      message.error(t("message.giveExpFail", { err: body?.error || JSON.stringify(body) || "" }));
    }
  } catch (e) {
    console.error("[GM] doGiveAncientTech error:", e);
    message.error("给予失败: " + e.message);
  }
};

const handleGmOp = async (key) => {
  if (["give_item", "give_pal", "tp", "give_custom_pal"].includes(key)) {
    // 获取在线玩家列表供传送/帕鲁给予使用
    try {
      const { data } = await new ApiService().getOnlinePlayerList();
      onlinePlayers.value = Array.isArray(data.value) ? data.value : [];
    } catch (_) {
      onlinePlayers.value = [];
    }
  }
  switch (key) {
    case "give_item":      showGiveItemModal.value = true; break;
    case "give_pal":       showGivePalModal.value = true; break;
    case "tp":             showTpModal.value = true; break;
    case "give_custom_pal": showCustomPalModal.value = true; break;
    case "give_exp":       showGiveExpModal.value = true; break;
    case "give_tech":      showGiveTechModal.value = true; break;
    case "give_ancient_tech": showGiveAncientTechModal.value = true; break;
    case "learn_tech":       showLearnTechModal.value = true; break;
    case "view_pals":      activeTab.value = t("item.palList"); break;
    case "view_items":     activeTab.value = t("item.itemList"); break;
  }
};

// 帕鲁列表子选项卡: backpack / palbox / basecamp
const palSubTab = ref("backpack");
const backpackPalsList = ref([]);
const backpackPalsLoading = ref(false);
const backpackPalsLoaded = ref(false);

// 从父组件 playerPalsList 中提取实时背包帕鲁（带 _backpack_live 标记）
const getLivePalsFromProp = () => {
  const all = playerPalsList.value || [];
  return all.filter(p => p._backpack_live === true);
};

const loadBackpackPals = async () => {
  // 如果父组件已经推送了实时数据，直接使用，不再重复调 API
  const livePals = getLivePalsFromProp();
  if (livePals.length > 0) {
    backpackPalsList.value = livePals;
    backpackPalsLoaded.value = true;
    return;
  }
  if (backpackPalsLoading.value) return;
  backpackPalsLoading.value = true;
  try {
    const res = await new ApiService().exportPlayerPals({ playerUid: playerInfo.value.player_uid });
    const { code, body } = parseRes(res);
    if (code === 200 && Array.isArray(body?.pals)) {
      backpackPalsList.value = body.pals;
    } else {
      backpackPalsList.value = [];
      if (code !== 200) message.error(body?.error || t("message.fail"));
    }
  } catch (e) {
    backpackPalsList.value = [];
    message.error(e.message);
  } finally {
    backpackPalsLoading.value = false;
    backpackPalsLoaded.value = true;
  }
};

const onPalSubTabChange = (tab) => {
  palSubTab.value = tab;
  searchValue.value = "";
  getPagination(tab).page = 1;
  if (tab === "backpack" && !backpackPalsLoaded.value) loadBackpackPals();
  mergeCurrentPalsList();
};

// ── Three source lists ────────────────────────────────────────────────────────
const getSavBackpackPalsList = () => {
  const all = playerPalsList.value || [];
  const hasFlag = all.some(p => p.in_palbox !== undefined);
  if (!hasFlag) return [];
  return all.filter(p => p.in_palbox === false);
};

const getPalboxList = () => {
  const all = playerPalsList.value || [];
  return all.filter(p => p.in_palbox === true && !p.is_base_pal);
};

const getBasecampList = () => {
  const all = playerPalsList.value || [];
  return all.filter(p => p.is_base_pal === true);
};

const mergeCurrentPalsList = () => {
  if (palSubTab.value === "backpack") {
    currentPalsList.value = backpackPalsLoaded.value
      ? [...backpackPalsList.value]
      : getSavBackpackPalsList();
  } else if (palSubTab.value === "palbox") {
    currentPalsList.value = getPalboxList();
  } else {
    currentPalsList.value = getBasecampList();
  }
};

// ── Independent pagination per tab ───────────────────────────────────────────
const makePagination = () => reactive({
  page: 1,
  pageSize: 25,
  showSizePicker: true,
  pageSizes: [25, 50, 100, 200],
  onChange: (page) => { getPagination(palSubTab.value).page = page; },
  onUpdatePageSize: (pageSize) => {
    getPagination(palSubTab.value).pageSize = pageSize;
    getPagination(palSubTab.value).page = 1;
  },
});
const paginationBackpack  = makePagination();
const paginationPalbox    = makePagination();
const paginationBasecamp  = makePagination();
const getPagination = (tab) =>
  tab === "backpack" ? paginationBackpack :
  tab === "palbox"   ? paginationPalbox   : paginationBasecamp;

// Legacy alias used by item tables
const paginationReactive = paginationBackpack;

// 帕鲁列表
const currentPalsList = ref([]);
const activeSkillById = new Map(activeSkills.map((skill) => [skill.id, skill]));
const passiveById = new Map(passiveSkills.map((skill) => [skill.id, skill]));
const elementMeta = {
  Normal: ["无", "#8b8f97"], Fire: ["火", "#e85d4a"], Water: ["水", "#4098fc"], Aqua: ["水", "#4098fc"],
  Ice: ["冰", "#67c8d5"], Leaf: ["草", "#36ad6a"], Earth: ["地", "#b68a5a"], Electric: ["雷", "#e6b422"],
  Electricity: ["雷", "#e6b422"], Thunder: ["雷", "#e6b422"], Dark: ["暗", "#8b65c2"], Dragon: ["龙", "#6f7de8"],
};
const passiveRankColors = { 5: "#e85d75", 4: "#f0a020", 3: "#8a62d3", 2: "#4098fc", 1: "#36ad6a", 0: "#8b8f97", "-1": "#8b8f97", "-2": "#d07836", "-3": "#d03050" };
const firstDefined = (row, keys, fallback = "—") => {
  for (const key of keys) if (row?.[key] !== undefined && row[key] !== null && row[key] !== "") return row[key];
  return fallback;
};
const normalizeList = (value) => Array.isArray(value) ? value : typeof value === "string" ? value.split(",").map((item) => item.trim()).filter(Boolean) : [];
const palType = (row) => firstDefined(row, ["PalID", "pal_id", "type"], "Unknown");
const palLevel = (row) => Number(firstDefined(row, ["Level", "level"], 0));
const palGender = (row) => firstDefined(row, ["Gender", "gender"], "");
const palNickname = (row) => firstDefined(row, ["Nickname", "nickname"], "");
const palLucky = (row) => Boolean(firstDefined(row, ["Shiny", "is_lucky"], false));
const palPassives = (row) => normalizeList(firstDefined(row, ["Passives", "passive_skills", "passiveSkills", "passives", "passive_skill_list", "skills"], []));
const palActiveSkills = (row) => normalizeList(firstDefined(row, ["ActiveSkills", "active_skills", "activeSkills", "active_skill_list", "wazas", "equip_waza", "equipWaza"], []));

const renderTooltipTag = (label, description, tagProps = {}) => h(NTooltip, { trigger: "hover", placement: "top", style: "max-width:360px" }, {
  trigger: () => h(NTag, { size: "small", ...tagProps }, { default: () => label }), default: () => description || "暂无说明",
});

const stripWazaPrefix = (skillId) => {
  if (typeof skillId !== "string") return skillId;
  const colonIdx = skillId.lastIndexOf(":");
  return colonIdx !== -1 ? skillId.slice(colonIdx + 1) : skillId;
};

const localizedActiveSkillName = (skillId) => {
  const bareId = stripWazaPrefix(skillId);
  const detail = activeSkillById.get(bareId);
  const language = String(locale.value || "zh").toLowerCase();
  const metadataName = language.startsWith("zh")
    ? detail?.zh
    : language.startsWith("ja")
      ? detail?.ja
      : detail?.name;
  const localizedName = localizedSkillName(bareId, locale.value, skillMap);
  return localizedName === bareId ? metadataName || bareId : localizedName;
};
const renderActiveSkill = (skillId) => {
  const bareId = stripWazaPrefix(skillId);
  const detail = activeSkillById.get(bareId);
  const [label, color] = elementMeta[detail?.element] || elementMeta.Normal;
  const language = String(locale.value || "zh").toLowerCase();
  const tooltipDesc = language.startsWith("zh")
    ? detail?.zh
      ? `${detail.zh}（${detail.name}）`
      : bareId
    : language.startsWith("ja")
      ? detail?.ja
        ? `${detail.ja}（${detail.name}）`
        : bareId
      : detail?.name || bareId;
  return h("span", { class: "pal-skill-wrap" }, [
    h("span", { class: "element-icon", style: { "--element-color": color }, title: detail?.element || "Normal" }, label),
    renderTooltipTag(localizedActiveSkillName(skillId), tooltipDesc, { bordered: true, style: { borderColor: color, color } }),
  ]);
};
const renderPassive = (passiveId) => {
  const info = passiveById.get(passiveId);
  const localized = passiveI18n?.[locale.value]?.[passiveId] || passiveI18n?.en?.[passiveId];
  const color = passiveRankColors[Number(info?.rank || 0)] || passiveRankColors[0];
  return renderTooltipTag(localized?.name || info?.zh || info?.name || passiveId, localized?.desc || passiveId, { bordered: true, style: { borderColor: color, color } });
};
const createPlayerPalsColumns = () => {
  return [
    {
      title: "",
      key: "",
      render(row) {
        return h(NAvatar, {
          size: "small",
          src: getPalAvatar(palType(row)),
          fallbackSrc: getUnknowPalAvatar(false),
        });
      },
    },
    {
      title: t("pal.type"),
      key: "type",
      // defaultSortOrder: 'ascend',
      sorter: "default",
      render(row) {
        return [
          h(
            NTag,
            {
              style: {
                marginRight: "6px",
              },
              type: palGender(row) == "Male" ? "primary" : "error",
              bordered: false,
            },
            {
              default: () => (palGender(row) == "Male" ? "♂" : "♀"),
            },
          ),
          h(
            "div",
            {
              style: {
                display: "inline-block",
                color: palLucky(row) ? "darkorange" : getDarkModeColor(),
                fontWeight: palLucky(row) ? "bold" : "normal",
              },
            },
            {
              default: () => getPalName(palType(row)),
            },
          ),
        ];
      },
    },
    {
      title: t("pal.level"),
      key: "level",
      width: 70,
      defaultSortOrder: "descend",
      sorter: "default",
      render(row) {
        return "Lv." + palLevel(row);
      },
    },
    {
      title: "主动技能",
      key: "skills",
      minWidth: 160,
      render(row) {
        return h("div", { class: "pal-tag-list" }, palActiveSkills(row).map(renderActiveSkill));
      },
      filterOptions: skillTypeList.value.map((value) => ({ label: value, value })),
      filter(value, row) {
        return palActiveSkills(row).some((skill) => localizedActiveSkillName(skill).includes(value));
      },
    },
    {
      title: "被动词条",
      key: "passives",
      minWidth: 150,
      render(row) {
        return h("div", { class: "pal-tag-list" }, palPassives(row).map(renderPassive));
      },
    },
    {
      title: "",
      key: "actions",
      render(row) {
        return h(NSpace, { size: "small" }, { default: () => [
          h(NButton, { size: "small", onClick: () => showPalDetail(row) }, { default: () => t("button.detail") }),
          h(NButton, { size: "small", type: "error", ghost: true, onClick: () => confirmDeletePal(row) }, { default: () => t("button.deletePal") }),
        ] });
      },
    },
  ];
};

// 统一解析 useFetch 响应
const parseRes = (res) => ({
  code: res.statusCode?.value ?? res.statusCode,
  body: res.data?.value ?? res.data,
});

// 删除帕鲁
const confirmDeletePal = (row) => {
  dialog.warning({
    title: t("message.deletePalTitle"),
    content: t("message.deletePalConfirm", { name: getPalName(palType(row)) }),
    positiveText: t("button.confirm"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      const payload = {
        playerUid: playerInfo.value.player_uid,
        pal_type: palType(row),
        level: palLevel(row),
        gender: palGender(row),
        is_lucky: palLucky(row),
      };
      console.log("[GM] releasePal payload:", payload);
      try {
        const res = await new ApiService().releasePal(payload);
        const { code, body } = parseRes(res);
        console.log("[GM] releasePal response:", code, body);
        if (code === 200) {
          message.success(t("message.deletePalSuccess", { msg: body?.message || "OK" }));
          currentPalsList.value = currentPalsList.value.filter(p => p !== row);
          setTimeout(() => emit("refresh-pals"), 500);
        } else {
          message.error(t("message.deletePalFail", { err: body?.error || JSON.stringify(body) || "" }));
        }
      } catch (e) {
        console.error("[GM] releasePal error:", e);
        message.error(t("message.deletePalFail", { err: e.message }));
      }
    },
  });
};

watch(
  () => playerPalsList.value,
  () => {
    // 父组件通过 _backpack_live 标记推送了实时背包帕鲁，直接同步
    const livePals = getLivePalsFromProp();
    if (livePals.length > 0) {
      backpackPalsList.value = livePals;
      backpackPalsLoaded.value = true;
    }
    paginationBackpack.page = 1;
    paginationPalbox.page = 1;
    paginationBasecamp.page = 1;
    searchValue.value = "";
    mergeCurrentPalsList();
    mergeItems();
  },
);

// 切换玩家时重置背包帕鲁状态
watch(
  () => playerInfo.value?.player_uid,
  () => {
    backpackPalsList.value = [];
    backpackPalsLoaded.value = false;
    palSubTab.value = "backpack";
    currentPalsList.value = [];
  },
);

// 切换到帕鲁列表主 tab 时自动加载背包帕鲁
watch(
  () => activeTab.value,
  (val) => {
    if (val && val === t("item.palList") && palSubTab.value === "backpack" && !backpackPalsLoaded.value) {
      loadBackpackPals();
    }
  },
);

// 背包帕鲁加载完成后同步到当前列表
watch(
  () => backpackPalsList.value,
  (newVal) => {
    if (palSubTab.value === "backpack") {
      currentPalsList.value = newVal || [];
    }
  },
);

const searchValue = ref("");
const getActiveSourceList = () => {
  if (palSubTab.value === "backpack") {
    return backpackPalsLoaded.value ? backpackPalsList.value : getSavBackpackPalsList();
  } else if (palSubTab.value === "palbox") {
    return getPalboxList();
  }
  return getBasecampList();
};
const clickSearch = () => {
  const pattern = /^\s*$|(\s)\1/;
  const source = getActiveSourceList();
  if (searchValue.value && !pattern.test(searchValue.value)) {
    currentPalsList.value = source.filter((item) => {
      return (
        palPassives(item).some((skill) => {
          return localizedSkillName(skill, locale.value, skillMap).includes(
            searchValue.value,
          );
        }) || getPalName(palType(item)).includes(searchValue.value)
      );
    });
  } else {
    currentPalsList.value = [...source];
  }
  getPagination(palSubTab.value).page = 1;
};
const clearSearch = () => {
  nextTick(() => {
    clickSearch();
  });
};

// 帕鲁详情
const showPalDetailModal = ref(false);
const palDetail = ref({});

// 将存档格式（小写字段）规范化为 PalDetail.vue 期待的大写格式
const normalizePalForDetail = (pal) => {
  if (!pal) return pal;
  // 如果已经是大写格式（背包帕鲁），直接返回
  if (pal.PalID !== undefined || pal.Level !== undefined) return pal;
  return {
    PalID:          pal.type ?? pal.pal_id ?? "",
    Nickname:       pal.nickname ?? "",
    Gender:         pal.gender ?? "",
    Level:          pal.level ?? 0,
    Exp:            pal.exp ?? 0,
    Shiny:          pal.is_lucky ?? false,
    IsBoss:         pal.is_boss ?? false,
    IsTower:        pal.is_tower ?? false,
    HP:             pal.hp ?? 0,
    CraftSpeed:     pal.workspeed ?? 0,
    IVs: {
      Health:       pal.talent_hp ?? 0,
      Attack:       pal.talent_shot ?? 0,
      Defense:      pal.talent_defense ?? 0,
    },
    PalSouls: (() => {
      const s = {};
      if (pal.rank       !== undefined) s.rank        = pal.rank;
      if (pal.rank_attack !== undefined) s.AttackMelee = pal.rank_attack;
      if (pal.rank_defence !== undefined) s.Defense    = pal.rank_defence;
      if (pal.rank_craftspeed !== undefined) s.CraftSpeed = pal.rank_craftspeed;
      if (pal.stars !== undefined) s.Stars = pal.stars;
      return s;
    })(),
    ActiveSkills:   Array.isArray(pal.active_skills) ? pal.active_skills : [],
    LearntSkills:   Array.isArray(pal.mastered_skills) ? pal.mastered_skills : [],
    Passives:       Array.isArray(pal.passive_skills) ? pal.passive_skills
                    : Array.isArray(pal.skills) ? pal.skills : [],
    in_palbox:      pal.in_palbox,
    is_base_pal:    pal.is_base_pal,
  };
};

const showPalDetail = (pal) => {
  palDetail.value = normalizePalForDetail(pal);
  showPalDetailModal.value = true;
};

// UID、Steam64 复制
const copyText = async (text) => {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text);
      message.success(t("message.copysuccess"));
    } catch (err) {
      message.error(t("message.copyerr", { err }));
    }
  } else {
    const textarea = document.createElement("textarea");
    textarea.value = text;
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand("copy");
      message.success(t("message.copysuccess"));
    } catch (err) {
      message.error(t("message.copyerr", { err }));
    }
    document.body.removeChild(textarea);
  }
};
// 查看公会
const toGuilds = async (uid) => {
  playerToGuildStore().setCurrentUid(uid);
  playerToGuildStore().setUpdateStatus("guilds");
};

// 加入白名单
const showAddWhiteListModal = ref(false);
const addWhiteData = ref({
  name: "",
  player_uid: "",
  steam_id: "",
});
const addWhiteList = async () => {
  const { data, statusCode } = await new ApiService().addWhitelist(
    addWhiteData,
  );
  if (statusCode.value === 200) {
    message.success(t("message.addwhitesuccess"));
    showAddWhiteListModal.value = false;
    await getWhiteList();
  } else {
    message.error(t("message.addwhitefail", { err: data.value?.error }));
  }
};
const handleAddWhiteList = () => {
  if (isLogin.value) {
    addWhiteData.value.name = playerInfo.value.nickname;
    addWhiteData.value.player_uid = playerInfo.value.player_uid;
    addWhiteData.value.steam_id = playerInfo.value.steam_id;
    showAddWhiteListModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showAddWhiteListModal.value = true;
  }
};
// 移除白名单
const removeWhitelist = async (player) => {
  if (isWhite(player)) {
    const { data, statusCode } = await new ApiService().removeWhitelist(player);
    if (statusCode.value === 200) {
      message.success(t("message.removewhitesuccess"));
      await getWhiteList();
    } else {
      message.error(t("message.removewhitefail", { err: data.value?.error }));
    }
  }
};

// 封禁、踢出
const handelPlayerAction = async (type) => {
  if (!isLogin.value) {
    message.error($t("message.requireauth"));
    showLoginModal.value = true;
    return;
  }
  const name = playerInfo?.value?.nickname || playerInfo?.value?.player_uid || "";
  const id = playerInfo?.value?.user_id || (playerInfo?.value?.steam_id ? `steam_${playerInfo.value.steam_id}` : playerInfo?.value?.player_uid) || "";
  const param = {
    ban: {
      title: t("message.bantitle"),
      content: t("message.banwarn", { name, id }),
    },
    unban: {
      title: t("message.unbantitle"),
      content: t("message.unbanwarn", { name, id }),
    },
    kick: {
      title: t("message.kicktitle"),
      content: t("message.kickwarn", { name, id }),
    },
  }[type];
  dialog.warning({
    ...param,
    positiveText: t("button.confirm"),
    negativeText: t("button.cancel"),
    onPositiveClick: async () => {
      if (type === "ban") {
        const { data, statusCode } = await new ApiService().banPlayer({
          playerUid: playerInfo?.value.player_uid,
        });
        if (statusCode.value === 200) {
          message.success(t("message.bansuccess"));
        } else {
          message.error(t("message.banfail", { err: data.value?.error }));
        }
      } else if (type === "unban") {
        const { data, statusCode } = await new ApiService().unbanPlayer({
          playerUid: playerInfo?.value.player_uid,
        });
        if (statusCode.value === 200) {
          message.success(t("message.unbansuccess"));
        } else {
          message.error(t("message.unbanfail", { err: data.value?.error }));
        }
      } else if (type === "kick") {
        const { data, statusCode } = await new ApiService().kickPlayer({
          playerUid: playerInfo?.value.player_uid,
        });
        if (statusCode.value === 200) {
          message.success(t("message.kicksuccess"));
        } else {
          message.error(t("message.kickfail", { err: data.value?.error }));
        }
      }
    },
  });
};

// 获取白名单列表
const whiteList = computed(() => whitelistStore().getWhitelist());
const getWhiteList = async () => {
  if (isLogin.value) {
    const { data, statusCode } = await new ApiService().getWhitelist();
    if (statusCode.value === 200) {
      if (data.value) {
        whitelistStore().setWhitelist(data.value);
      } else {
        whitelistStore().setWhitelist([]);
      }
    }
  }
};

// 是否在白名单中
const isWhite = (player) => {
  if (whiteList.value.length === 0) {
    return false;
  }
  return whiteList.value.some((whitelistItem) => {
    return (
      (whitelistItem.player_uid &&
        whitelistItem.player_uid === player.player_uid) ||
      (whitelistItem.steam_id && whitelistItem.steam_id === player.steam_id)
    );
  });
};

onMounted(async () => {
  skillTypeList.value = getSkillTypeList();
  await getWhiteList();
  localeLowerPalMap.value = Object.keys(palMap[locale.value]).reduce(
    (acc, key) => {
      acc[key.toLowerCase()] = palMap[locale.value][key];
      return acc;
    },
    {},
  );
});

// 其他操作
const getDarkModeColor = () => {
  return isDarkMode.value ? "#fff" : "#000";
};

const getSkillTypeList = () => {
  if (skillMap[locale.value]) {
    return Object.values(skillMap[locale.value]).map((item) => item.name);
  } else {
    return [];
  }
};
const getStatusPointLabel = (rawKey) => {
  const translationKey = statusPointTranslationKey(rawKey);
  return translationKey ? t(`statusPoint.${translationKey}`) : rawKey;
};
const getPalAvatar = (name) => {
  const lowerName = name.toLowerCase();
  return new URL(`../../../assets/pals/${lowerName}.png`, import.meta.url).href;
};
const getPalName = (name) => {
  const lowerName = name.toLowerCase();
  return localeLowerPalMap.value[lowerName]
    ? localeLowerPalMap.value[lowerName]
    : name;
};
const getItemIcon = (id) => {
  return new URL(`../../../assets/items/${id}.webp`, import.meta.url).href;
};
const getUnknowPalAvatar = (is_boss = false) => {
  if (is_boss) {
    return new URL("@/assets/pals/boss_unknown.png", import.meta.url).href;
  }
  return new URL("@/assets/pals/unknown.png", import.meta.url).href;
};
const getPlatformColor = (userId) => {
  if (!userId) return platformColors.default;
  return platformColors[userId.split("_")[0]] || platformColors.default;
};
const displayLastOnline = (lastOnline) => {
  if (dayjs(lastOnline).year() < 1970) return "Unknown";
  return dayjs(lastOnline).format("YYYY-MM-DD HH:mm:ss");
};

const displayHP = (hp, max_hp) => {
  return (hp / 1000).toFixed(0) + "/" + (max_hp / 1000).toFixed(0);
};

const percentageHP = (hp, max_hp) => {
  if (max_hp === 0) {
    return 0;
  }
  return ((hp / max_hp) * 100).toFixed(2);
};

const mergedItems = ref({});
const mergeItems = () => {
  mergedItems.value = {};

  if (!playerInfo.value.items) return;
  for (const [containerId, items] of Object.entries(playerInfo.value.items)) {
    mergedItems.value[containerId] = items.map((item) => {
      const frontendItem = palItems[locale.value].find(
        (frontItem) => frontItem.id === item.ItemId,
      );
      if (!frontendItem) {
        return {
          ...item,
          id: item.ItemId,
          name: item.ItemId,
          description: "No description.",
          key: item.ItemId,
        };
      }
      return {
        ...item,
        id: frontendItem.id,
        name: frontendItem.name,
        description: frontendItem.description,
        key: frontendItem.key,
      };
    });
  }
};

const editingItemKey = ref(null);
const editingItemCount = ref(1);

// 删除物品弹窗状态
const deleteItemModal = ref(false);
const deleteItemTarget = ref(null);
const deleteItemAmount = ref(1);

const startEditItem = (row) => {
  editingItemKey.value = row.ContainerId + "_" + row.SlotIndex;
  editingItemCount.value = row.StackCount;
};

const cancelEditItem = () => {
  editingItemKey.value = null;
};

const saveItemCount = async (row) => {
  const diff = editingItemCount.value - row.StackCount;
  if (diff === 0) { cancelEditItem(); return; }
  try {
    let res;
    if (diff > 0) {
      const payload = { playerUid: playerInfo.value.player_uid, item_id: row.id, amount: diff };
      console.log("[GM] giveItem payload:", payload);
      res = await new ApiService().giveItem(payload);
    } else {
      const payload = { playerUid: playerInfo.value.player_uid, item_id: row.id, amount: Math.abs(diff) };
      console.log("[GM] deleteItem payload:", payload);
      res = await new ApiService().deleteItem(payload);
    }
    const { code, body } = parseRes(res);
    console.log("[GM] saveItemCount response:", code, body);
    if (code === 200) {
      row.StackCount = editingItemCount.value;
      message.success(t("message.countUpdated", { msg: body?.message || "OK" }));
      cancelEditItem();
    } else {
      message.error(t("message.countUpdateFail", { err: body?.error || JSON.stringify(body) || "" }));
    }
  } catch (e) {
    console.error("[GM] saveItemCount error:", e);
    message.error(t("message.countUpdateFail", { err: e.message }));
  }
};

const openDeleteItemModal = (row) => {
  deleteItemTarget.value = row;
  deleteItemAmount.value = row.StackCount;
  deleteItemModal.value = true;
};

const doDeleteItem = async () => {
  const row = deleteItemTarget.value;
  if (!row) return;
  const amount = deleteItemAmount.value;
  const payload = { playerUid: playerInfo.value.player_uid, item_id: row.id, amount };
  console.log("[GM] deleteItem payload:", payload);
  try {
    const res = await new ApiService().deleteItem(payload);
    const { code, body } = parseRes(res);
    console.log("[GM] deleteItem response:", code, body);
    if (code === 200) {
      message.success(t("message.deleteItemOk", { msg: body?.message || "OK" }));
      if (amount >= row.StackCount) {
        for (const key of Object.keys(mergedItems.value)) {
          mergedItems.value[key] = mergedItems.value[key].filter(
            (i) => !(i.id === row.id && i.SlotIndex === row.SlotIndex && i.ContainerId === row.ContainerId)
          );
        }
      } else {
        row.StackCount -= amount;
      }
      deleteItemModal.value = false;
    } else {
      message.error(t("message.deleteItemErr", { err: body?.error || JSON.stringify(body) || "" }));
    }
  } catch (e) {
    console.error("[GM] deleteItem error:", e);
    message.error("删除失败: " + e.message);
  }
};

const createPlayerItemsColumns = () => {
  return [
    {
      title: "",
      key: "",
      width: 48,
      render(row) {
        return h(NAvatar, {
          size: "small",
          src: getItemIcon(row.id),
          fallbackSrc: getUnknowPalAvatar(),
        });
      },
    },
    {
      title: t("item.name"),
      key: "name",
    },
    {
      title: t("item.count"),
      key: "StackCount",
      width: 180,
      defaultSortOrder: "descend",
      sorter: "default",
      render(row) {
        const rowKey = row.ContainerId + "_" + row.SlotIndex;
        if (editingItemKey.value === rowKey) {
          return h(NInputNumber, {
            value: editingItemCount.value,
            min: 1,
            max: 9999,
            size: "small",
            style: "width:120px",
            onUpdateValue: (v) => { if (v != null) editingItemCount.value = v; },
          });
        }
        return String(row.StackCount);
      },
    },
    {
      title: "",
      key: "actions",
      width: 160,
      render(row) {
        const rowKey = row.ContainerId + "_" + row.SlotIndex;
        const isEditing = editingItemKey.value === rowKey;
        return h(NSpace, { size: "small" }, {
          default: () => isEditing ? [
            h(NButton, { size: "small", type: "primary", ghost: true, onClick: () => saveItemCount(row) },
              { default: () => t("button.save") }),
            h(NButton, { size: "small", onClick: cancelEditItem },
              { default: () => t("button.cancel") }),
          ] : [
            h(NButton, { size: "small", ghost: true, onClick: () => startEditItem(row) },
              { default: () => t("button.edit") }),
            h(NButton, { size: "small", type: "error", ghost: true, onClick: () => openDeleteItemModal(row) },
              { default: () => t("button.delete") }),
          ],
        });
      },
    },
  ];
};
</script>

<template>
  <div class="player-detail" :class="{ 'is-dark': isDarkMode }">
    <n-card
      content-style="padding: 24px 28px 36px;"
      id="player-info"
      :bordered="false"
      v-if="playerInfo?.nickname"
    >
      <section class="player-overview" aria-labelledby="player-detail-title">
        <div class="overview-heading">
          <div class="player-identity">
            <div class="player-title-row">
              <h1 id="player-detail-title" class="player-title">
                {{ playerInfo?.nickname }}
              </h1>
              <n-tag type="primary" size="large" round strong>
                Lv.{{ playerInfo?.level }}
                <template #icon>
                  <n-icon :component="CrownFilled" />
                </template>
              </n-tag>
            </div>
            <div class="identity-tags">
              <n-tag
                :bordered="false"
                :type="
                  props.isOnline ? 'success' : 'error'
                "
                size="small"
                round
              >
                {{
                  props.isOnline
                    ? $t("status.online")
                    : $t("status.offline")
                }}
              </n-tag>
              <n-tag
                v-if="playerInfo?.user_id"
                :bordered="false"
                round
                size="small"
                :color="getPlatformColor(playerInfo.user_id)"
              >
                {{ playerInfo.user_id.split("_")[0] }}
              </n-tag>
              <n-tag
                v-if="isWhite(playerInfo)"
                :bordered="false"
                round
                size="small"
                :color="{
                  color: isDarkMode ? '#fff' : '#d9c36c',
                  textColor: isDarkMode ? '#d9c36c' : '#fff',
                }"
              >
                {{ $t("status.whitelist") }}
              </n-tag>
              <span class="last-online-text">
                {{ $t("status.last_online") }}
                {{ displayLastOnline(playerInfo?.last_online) }}
              </span>
            </div>
          </div>
          <div class="heading-right-actions">
            <n-dropdown
              v-if="isLogin"
              trigger="click"
              :options="gmOpsOptions"
              @select="handleGmOp"
            >
              <n-button type="primary" secondary strong>
                {{ $t("button.actions") }}
                <template #icon>
                  <n-icon>
                    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512"><path fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="48" d="M184 112l144 144l-144 144"></path></svg>
                  </n-icon>
                </template>
              </n-button>
            </n-dropdown>
            <n-button
              @click="toGuilds(playerInfo?.player_uid)"
              type="warning"
              secondary
              strong
            >
              {{ $t("button.viewGuild") }}
              <template #icon>
                <n-icon><PersonSearchSharp /></n-icon>
              </template>
            </n-button>
          </div>
        </div>

        <div v-if="isLogin" class="identity-copy-list">
          <n-button
            class="identity-copy"
            secondary
            @click="copyText(playerInfo?.player_uid)"
          >
            <span class="copy-label">UID</span>
            <span class="copy-value">{{ playerInfo?.player_uid }}</span>
            <template #icon>
              <n-icon><ContentCopyFilled /></n-icon>
            </template>
          </n-button>
          <n-button
            class="identity-copy"
            secondary
            @click="copyText(playerInfo?.steam_id)"
          >
            <span class="copy-label">Steam64</span>
            <span class="copy-value">
              {{ playerInfo.steam_id ? playerInfo.steam_id : "--" }}
            </span>
            <template #icon>
              <n-icon><ContentCopyFilled /></n-icon>
            </template>
          </n-button>
        </div>

        <div
          v-if="
            playerInfo.ip ||
            playerInfo.ping ||
            playerInfo.location_x ||
            playerInfo.location_y
          "
          class="runtime-meta"
        >
          <span v-if="playerInfo.ip">IP {{ playerInfo.ip }}</span>
          <span v-if="playerInfo.ping">
            Ping {{ playerInfo.ping.toFixed(2) }}
          </span>
          <span v-if="playerInfo.location_x"
            >X {{ playerInfo.location_x }}</span
          >
          <span v-if="playerInfo.location_y"
            >Y {{ playerInfo.location_y }}</span
          >
        </div>

        <div class="status-grid">
          <div
            v-for="status in Object.entries(playerInfo?.status_point || {})"
            :key="status[0]"
            class="status-card"
          >
            <span
              class="status-label"
              :title="getStatusPointLabel(status[0])"
              >{{ getStatusPointLabel(status[0]) }}</span
            >
            <strong class="status-value">{{ status[1] }}</strong>
          </div>
        </div>
      </section>
      <!-- <n-space vertical>
        <n-progress
          type="line"
          status="error"
          indicator-placement="inside"
          :percentage="percentageHP(playerInfo?.hp, playerInfo?.max_hp)"
          :height="24"
          :border-radius="4"
          :fill-border-radius="0"
          >HP: {{ displayHP(playerInfo?.hp, playerInfo?.max_hp) }}</n-progress
        >
        <n-progress
          type="line"
          indicator-placement="inside"
          :percentage="
            percentageHP(playerInfo?.shield_hp, playerInfo?.shield_max_hp)
          "
          :height="24"
          :border-radius="4"
          :fill-border-radius="0"
          >SHIELD:
          {{
            displayHP(playerInfo?.shield_hp, playerInfo?.shield_max_hp)
          }}</n-progress
        >
      </n-space> -->
      <div v-if="isLogin" class="detail-tabs">
        <n-tabs v-model:value="activeTab" type="line" size="large" animated>
          <n-tab-pane :name="$t('item.palList')">
            <n-tabs
              v-model:value="palSubTab"
              type="segment"
              animated
              class="mt-3"
              @update:value="onPalSubTabChange"
            >
              <!-- 背包帕鲁 -->
              <n-tab-pane name="backpack" tab="背包帕鲁">
                <div class="w-full mt-3">
                  <n-input-group class="w-full flex justify-end">
                    <n-input v-model:value="searchValue" clearable
                      :placeholder="$t('input.searchPlaceholder')"
                      :on-clear="clearSearch" @keydown.enter="clickSearch" />
                    <n-button type="primary" class="w-20" @click="clickSearch">{{ $t("button.search") }}</n-button>
                  </n-input-group>
                </div>
                <n-spin :show="backpackPalsLoading">
                  <n-data-table class="mt-2" size="small"
                    :columns="createPlayerPalsColumns()"
                    :data="currentPalsList" :bordered="false" striped
                    :pagination="paginationBackpack" />
                </n-spin>
              </n-tab-pane>
              <!-- 终端帕鲁 -->
              <n-tab-pane name="palbox" tab="终端帕鲁">
                <div class="w-full mt-3">
                  <n-input-group class="w-full flex justify-end">
                    <n-input v-model:value="searchValue" clearable
                      :placeholder="$t('input.searchPlaceholder')"
                      :on-clear="clearSearch" @keydown.enter="clickSearch" />
                    <n-button type="primary" class="w-20" @click="clickSearch">{{ $t("button.search") }}</n-button>
                  </n-input-group>
                </div>
                <n-data-table class="mt-2" size="small"
                  :columns="createPlayerPalsColumns()"
                  :data="currentPalsList" :bordered="false" striped
                  :pagination="paginationPalbox" />
              </n-tab-pane>
              <!-- 据点帕鲁 -->
              <n-tab-pane name="basecamp" tab="据点帕鲁">
                <div class="w-full mt-3">
                  <n-input-group class="w-full flex justify-end">
                    <n-input v-model:value="searchValue" clearable
                      :placeholder="$t('input.searchPlaceholder')"
                      :on-clear="clearSearch" @keydown.enter="clickSearch" />
                    <n-button type="primary" class="w-20" @click="clickSearch">{{ $t("button.search") }}</n-button>
                  </n-input-group>
                </div>
                <n-data-table class="mt-2" size="small"
                  :columns="createPlayerPalsColumns()"
                  :data="currentPalsList" :bordered="false" striped
                  :pagination="paginationBasecamp" />
              </n-tab-pane>
            </n-tabs>
          </n-tab-pane>
          <n-tab-pane :name="$t('item.itemList')">
            <n-tabs type="segment" animated>
              <n-tab-pane :name="$t('item.commonContainer')">
                <n-data-table
                  size="small"
                  :columns="createPlayerItemsColumns()"
                  :data="mergedItems['CommonContainerId']"
                  :bordered="false"
                  striped
                  :pagination="paginationReactive"
                />
              </n-tab-pane>
              <n-tab-pane :name="$t('item.essentialContainer')">
                <n-data-table
                  size="small"
                  :columns="createPlayerItemsColumns()"
                  :data="mergedItems['EssentialContainerId']"
                  :bordered="false"
                  striped
                  :pagination="paginationReactive"
                />
              </n-tab-pane>
              <n-tab-pane :name="$t('item.weaponContainer')">
                <n-data-table
                  size="small"
                  :columns="createPlayerItemsColumns()"
                  :data="mergedItems['WeaponLoadOutContainerId']"
                  :bordered="false"
                  striped
                />
              </n-tab-pane>
              <n-tab-pane :name="$t('item.armorContainer')">
                <n-data-table
                  class="mt-1"
                  size="small"
                  :columns="createPlayerItemsColumns()"
                  :data="mergedItems['PlayerEquipArmorContainerId']"
                  :bordered="false"
                  striped
                />
              </n-tab-pane>
            </n-tabs>
          </n-tab-pane>
        </n-tabs>
      </div>
    </n-card>
    <!-- 加入白名单，封禁，踢出 -->
    <n-flex
      justify="end"
      class="player-actions"
      v-if="playerInfo?.nickname && isLogin"
    >
      <n-button
        @click="
          isWhite(playerInfo)
            ? removeWhitelist(playerInfo)
            : handleAddWhiteList()
        "
        :type="isWhite(playerInfo) ? 'warning' : 'success'"
        size="large"
        secondary
        strong
        round
      >
        <template #icon>
          <n-icon>
            <ShieldCheckmarkOutline />
          </n-icon>
        </template>
        {{
          isWhite(playerInfo)
            ? $t("button.removeWhitelist")
            : $t("button.joinWhitelist")
        }}
      </n-button>
      <n-button
        @click="handelPlayerAction('ban')"
        type="error"
        size="large"
        secondary
        strong
        round
      >
        <template #icon>
          <n-icon>
            <Ban />
          </n-icon>
        </template>
        {{ $t("button.ban") }}
      </n-button>
      <n-button
        @click="handelPlayerAction('unban')"
        type="success"
        size="large"
        secondary
        strong
        round
      >
        <template #icon>
          <n-icon>
            <Ban />
          </n-icon>
        </template>
        {{ $t("button.unban") }}
      </n-button>
      <n-button
        @click="handelPlayerAction('kick')"
        type="warning"
        size="large"
        secondary
        strong
        round
      >
        <template #icon>
          <n-icon>
            <LogOut />
          </n-icon>
        </template>
        {{ $t("button.kick") }}
      </n-button>
    </n-flex>
  </div>
  <!-- 帕鲁详情 modal -->
  <n-modal
    v-model:show="showPalDetailModal"
    preset="card"
    :style="{ width: '94%', maxWidth: '960px' }"
    header-style="padding:12px 20px;"
    content-style="padding:12px 20px;margin:0;"
    size="huge"
    :bordered="false"
    :segmented="{ content: 'soft', footer: 'soft' }"
  >
    <template #header-extra>
      <div class="flex pr-3 space-x-2">
        <n-tag type="primary" round> Lv.{{ palLevel(palDetail) }} </n-tag>
        <n-tag :type="palGender(palDetail) === 'Male' ? 'primary' : 'error'" round>
          {{ palGender(palDetail) === "Male" ? "♂" : "♀" }}
        </n-tag>
      </div>
    </template>
    <template #header>
      {{
        palNickname(palDetail) == ""
          ? getPalName(palType(palDetail))
          : palNickname(palDetail) + "(" + getPalName(palType(palDetail)) + ")"
      }}
    </template>
    <pal-detail :palDetail="palDetail"></pal-detail>
  </n-modal>

  <!-- 添加白名单 modal -->
  <n-modal
    v-model:show="showAddWhiteListModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 700px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('modal.addWhitelist')"
    :bordered="false"
  >
    <n-grid class="flex items-center">
      <n-gi span="5">
        <div class="flex justify-center">
          {{ $t("message.selectVerify") }}
        </div>
      </n-gi>
      <n-gi span="19">
        <n-input-group>
          <n-input
            v-model:value="addWhiteData.name"
            :style="{ width: '33%' }"
            :placeholder="$t('input.nickname')"
          />
          <n-input
            v-model:value="addWhiteData.player_uid"
            :style="{ width: '33%' }"
            :placeholder="$t('input.player_uid')"
          />
          <n-input
            v-model:value="addWhiteData.steam_id"
            :style="{ width: '33%' }"
            :placeholder="$t('input.steam_id')"
          />
        </n-input-group>
      </n-gi>
    </n-grid>
    <template #footer>
      <div class="flex justify-end">
        <n-button
          type="tertiary"
          @click="
            () => {
              showAddWhiteListModal = false;
            }
          "
        >
          {{ $t("button.cancel") }}
        </n-button>
        <n-button
          class="ml-3 w-40"
          type="primary"
          @click="addWhiteList"
          :disabled="
            !addWhiteData.name ||
            (!addWhiteData.player_uid && !addWhiteData.steam_id)
          "
        >
          {{ $t("button.confirm") }}
        </n-button>
      </div>
    </template>
  </n-modal>

  <!-- 给予道具 modal -->
  <n-modal
    v-model:show="showGiveItemModal"
    preset="card"
    style="width: 96%; max-width: 860px"
    :title="$t('button.giveItem')"
    header-style="padding: 12px 20px;"
    content-style="padding: 12px 20px;"
    :bordered="false"
  >
    <GmItemPicker
      v-if="showGiveItemModal && playerInfo?.player_uid"
      :player-uid="playerInfo.player_uid"
    />
  </n-modal>

  <!-- 给予帕鲁 modal -->
  <n-modal
    v-model:show="showGivePalModal"
    preset="card"
    style="width: 96%; max-width: 860px"
    :title="$t('button.givePal')"
    header-style="padding: 12px 20px;"
    content-style="padding: 12px 20px;"
    :bordered="false"
  >
    <GmPalPicker
      v-if="showGivePalModal && playerInfo?.player_uid"
      :player-uid="playerInfo.player_uid"
      :online-players="onlinePlayers"
    />
  </n-modal>

  <!-- 传送 modal -->
  <GmTeleportModal
    v-model:show="showTpModal"
    :player-uid="playerInfo?.player_uid || ''"
    :user-id="playerInfo?.user_id || (playerInfo?.steam_id ? 'steam_' + playerInfo.steam_id : '')"
    :online-players="onlinePlayers"
    @done="showTpModal = false"
  />

  <!-- 给予自定义帕鲁 modal -->
  <GmCustomPalModal
    v-model:show="showCustomPalModal"
    :player-uid="playerInfo?.player_uid || ''"
    :online-players="onlinePlayers"
  />

  <!-- 给予经验值 modal -->
  <n-modal v-model:show="showGiveExpModal" preset="card" style="width:90%;max-width:400px"
    :title="$t('button.giveExp')" :bordered="false" header-style="padding:12px 20px" content-style="padding:12px 20px">
    <n-form-item :label="$t('message.expAmount')" label-placement="top">
      <n-input-number v-model:value="giveExpAmount" :min="1" :max="999999999" style="width:100%" />
    </n-form-item>
    <template #footer>
      <n-flex justify="end" gap="8">
        <n-button @click="showGiveExpModal = false">{{ $t("button.cancel") }}</n-button>
        <n-button type="primary" @click="doGiveExp">{{ $t("message.confirmGiveBtn") }}</n-button>
      </n-flex>
    </template>
  </n-modal>

  <!-- 给予科技点数 modal -->
  <n-modal v-model:show="showGiveTechModal" preset="card" style="width:90%;max-width:400px"
    :title="$t('button.giveTech')" :bordered="false" header-style="padding:12px 20px" content-style="padding:12px 20px">
    <n-form-item :label="$t('message.techAmount')" label-placement="top">
      <n-input-number v-model:value="giveTechAmount" :min="1" :max="99999" style="width:100%" />
    </n-form-item>
    <template #footer>
      <n-flex justify="end" gap="8">
        <n-button @click="showGiveTechModal = false">{{ $t("button.cancel") }}</n-button>
        <n-button type="primary" @click="doGiveTech">{{ $t("message.confirmGiveBtn") }}</n-button>
      </n-flex>
    </template>
  </n-modal>

  <!-- 给予古代科技点数 modal -->
  <n-modal v-model:show="showGiveAncientTechModal" preset="card" style="width:90%;max-width:400px"
    :title="$t('button.giveAncientTech')" :bordered="false" header-style="padding:12px 20px" content-style="padding:12px 20px">
    <n-form-item :label="$t('message.ancientTechAmount')" label-placement="top">
      <n-input-number v-model:value="giveAncientTechAmount" :min="1" :max="99999" style="width:100%" />
    </n-form-item>
    <template #footer>
      <n-flex justify="end" gap="8">
        <n-button @click="showGiveAncientTechModal = false">{{ $t("button.cancel") }}</n-button>
        <n-button type="primary" @click="doGiveAncientTech">{{ $t("message.confirmGiveBtn") }}</n-button>
      </n-flex>
    </template>
  </n-modal>

  <!-- 学习指定科技 modal -->
  <n-modal v-model:show="showLearnTechModal" preset="card" style="width:96%;max-width:800px"
    :title="$t('button.learnTech')" :bordered="false" header-style="padding:12px 20px" content-style="padding:12px 20px">
    <GmTechPicker
      v-if="showLearnTechModal && playerInfo?.player_uid"
      :player-uid="playerInfo.player_uid"
    />
  </n-modal>

  <!-- 删除物品 modal -->
  <n-modal v-model:show="deleteItemModal" preset="card" style="width:90%;max-width:420px"
    :title="$t('message.deleteItemTitle', { name: deleteItemTarget?.name || '' })"
    :bordered="false" header-style="padding:12px 20px" content-style="padding:12px 20px">
    <n-form-item label-placement="top">
      <template #label>
        {{ $t("message.deleteItemLabel", { count: deleteItemTarget?.StackCount ?? 0 }) }}
      </template>
      <n-input-number
        v-model:value="deleteItemAmount"
        :min="1"
        :max="deleteItemTarget?.StackCount ?? 9999"
        style="width:100%"
      />
    </n-form-item>
    <template #footer>
      <n-flex justify="end" gap="8">
        <n-button @click="deleteItemModal = false">{{ $t("button.cancel") }}</n-button>
        <n-button type="error" @click="doDeleteItem">{{ $t("message.confirmDeleteBtn") }}</n-button>
      </n-flex>
    </template>
  </n-modal>
</template>

<style scoped lang="less">
.player-detail {
  min-height: 100%;
}

.player-overview {
  padding: 22px;
  border: 1px solid rgba(64, 152, 252, 0.22);
  border-radius: 16px;
  background: linear-gradient(
    135deg,
    rgba(64, 152, 252, 0.1),
    rgba(64, 152, 252, 0.025) 58%,
    transparent
  );
}

.is-dark .player-overview {
  border-color: rgba(64, 152, 252, 0.28);
  background: linear-gradient(
    135deg,
    rgba(64, 152, 252, 0.16),
    rgba(64, 152, 252, 0.045) 58%,
    transparent
  );
}

.overview-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
}

.heading-right-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.player-identity {
  min-width: 0;
}

.player-title-row,
.identity-tags,
.runtime-meta,
.identity-copy-list {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

.player-title-row {
  gap: 12px;
}

.player-title {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: clamp(24px, 2.2vw, 30px);
  font-weight: 700;
  line-height: 1.25;
  letter-spacing: -0.02em;
}

.identity-tags {
  gap: 8px;
  margin-top: 10px;
}

.last-online-text {
  color: rgba(24, 24, 28, 0.52);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.is-dark .last-online-text {
  color: rgba(255, 255, 255, 0.5);
}

.identity-copy-list {
  gap: 8px;
  margin-top: 18px;
}

.identity-copy {
  max-width: min(100%, 420px);
}

.copy-label {
  flex: none;
  font-weight: 650;
}

.copy-value {
  min-width: 0;
  overflow: hidden;
  color: rgba(24, 24, 28, 0.58);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.is-dark .copy-value {
  color: rgba(255, 255, 255, 0.54);
}

.runtime-meta {
  gap: 8px;
  margin-top: 12px;
}

.runtime-meta span {
  padding: 4px 9px;
  border-radius: 999px;
  background: rgba(24, 24, 28, 0.06);
  color: rgba(24, 24, 28, 0.62);
  font-size: 12px;
}

.is-dark .runtime-meta span {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.62);
}

.status-grid {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 10px;
  margin-top: 20px;
}

.status-card {
  min-width: 0;
  padding: 13px 14px;
  border: 1px solid rgba(24, 24, 28, 0.06);
  border-radius: 11px;
  background: rgba(255, 255, 255, 0.7);
}

.is-dark .status-card {
  border-color: rgba(255, 255, 255, 0.07);
  background: rgba(255, 255, 255, 0.055);
}

.status-label {
  display: -webkit-box;
  min-height: 32px;
  overflow: hidden;
  color: rgba(24, 24, 28, 0.52);
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.is-dark .status-label {
  color: rgba(255, 255, 255, 0.5);
}

.status-value {
  display: block;
  margin-top: 6px;
  font-size: 22px;
  font-weight: 650;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.detail-tabs {
  margin-top: 20px;
}

.pal-tag-list { display: flex; align-items: center; flex-wrap: wrap; gap: 3px; padding: 1px 0; }
.pal-tag-list :deep(.n-tag) { --n-height: 22px !important; padding: 0 5px; font-size: 11px; }
.pal-skill-wrap { display: inline-flex; align-items: center; gap: 2px; }
.element-icon { width: 18px; height: 18px; display: inline-flex; align-items: center; justify-content: center; flex: none; border: 1px solid var(--element-color); border-radius: 50%; background: rgba(64, 152, 252, 0.1); color: var(--element-color); font-size: 10px; font-weight: 700; }
.pal-stat-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 4px 8px; }
.pal-stat-grid span { display: flex; justify-content: space-between; gap: 5px; padding: 3px 6px; border-radius: 5px; background: rgba(64, 152, 252, 0.07); }
.pal-stat-grid small { color: rgba(24, 24, 28, 0.52); }
.is-dark .pal-stat-grid small { color: rgba(255, 255, 255, 0.52); }
.pal-stat-grid b { font-variant-numeric: tabular-nums; }

.player-actions {
  position: sticky;
  bottom: 0;
  z-index: 5;
  padding: 12px 20px;
  border-top: 1px solid rgba(24, 24, 28, 0.08);
  background: rgba(255, 255, 255, 0.94);
  backdrop-filter: blur(12px);
}

.is-dark .player-actions {
  border-top-color: rgba(255, 255, 255, 0.08);
  background: rgba(24, 24, 28, 0.94);
}

@media (max-width: 1100px) {
  .status-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .overview-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .heading-right-actions {
    align-self: flex-start;
  }
}
</style>
