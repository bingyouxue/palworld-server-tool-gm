<script setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import skillMap from "@/assets/skill.json";
import palMap from "@/assets/pal.json";
import activeSkills from "@/assets/gm/activeSkills.json";
import passives from "@/assets/gm/passives.json";
import passiveI18n from "@/assets/gm/passiveI18n.json";

const props = defineProps({ palDetail: { type: Object, default: () => ({}) } });
const { locale } = useI18n();
const activeSkillById = new Map(activeSkills.map((skill) => [skill.id, skill]));
const passiveById = new Map(passives.map((passive) => [passive.id, passive]));
const elementColors = { Normal: "#8b8f97", Fire: "#e85d4a", Water: "#4098fc", Aqua: "#4098fc", Ice: "#67c8d5", Leaf: "#36ad6a", Earth: "#b68a5a", Electric: "#e6b422", Electricity: "#e6b422", Thunder: "#e6b422", Dark: "#8b65c2", Dragon: "#6f7de8" };
const elementIcons = { Normal: "○", Fire: "火", Water: "水", Aqua: "水", Ice: "冰", Leaf: "草", Earth: "地", Electric: "雷", Electricity: "雷", Thunder: "雷", Dark: "暗", Dragon: "龙" };
const passiveRankColors = { 5: "#e85d75", 4: "#f0a020", 3: "#8a62d3", 2: "#4098fc", 1: "#36ad6a", 0: "#8b8f97", "-1": "#8b8f97", "-2": "#d07836", "-3": "#d03050" };

const text = {
  zh: {
    sections: { basic: "基础信息", status: "当前状态", souls: "灵魂强化", ivs: "个体潜力", active: "主动技能", learnt: "已学技能", passives: "被动技能", work: "额外工作适应性", disabled: "禁用工作偏好", other: "其他信息" },
    fields: { PalID: "帕鲁种类", Nickname: "昵称", SkinId: "外观", Gender: "性别", Level: "等级", Exp: "经验值", Shiny: "闪光", PartnerSkillLevel: "伙伴技能等级", CondensedPals: "浓缩数量", UnusedStatusPoints: "未使用属性点", FriendshipPoints: "亲密度", IsAwakening: "觉醒", PhysicalHealth: "身体状态", WorkerSick: "疾病状态", ImportedCharacter: "导入角色", HP: "血量", SP: "体力", MP: "能量", Hunger: "饱食度", MaxHunger: "最大饱食度", SAN: "SAN值", Support: "支援力", CraftSpeed: "工作速度", Health: "生命", Attack: "攻击", Defense: "防御", AttackMelee: "近战攻击", AttackShot: "远程攻击", BaseCampBattle: "据点战斗", Anyone: "任意工作" },
    work: { EmitFlame: "生火", Watering: "浇水", Seeding: "播种", GenerateElectricity: "发电", Handcraft: "手工作业", Collection: "采集", Deforest: "伐木", Mining: "采矿", OilExtraction: "采油", ProductMedicine: "制药", Cool: "冷却", Transport: "搬运", MonsterFarm: "牧场" },
    values: { true: "是", false: "否", Male: "雄性", Female: "雌性", None: "无", Healthful: "健康", Cold: "感冒", Sprain: "扭伤", Ulcer: "胃溃疡", Fracture: "骨折", Weakness: "虚弱", Depression: "抑郁", Bulimia: "暴食症", Hungry: "饥饿", Starving: "濒临饿死", Injured: "受伤", Sick: "生病" }, elements: { Normal: "无", Fire: "火", Water: "水", Aqua: "水", Ice: "冰", Leaf: "草", Earth: "地", Electric: "雷", Electricity: "雷", Thunder: "雷", Dark: "暗", Dragon: "龙" }, rank: "等级", empty: "无",
  },
  en: {
    sections: { basic: "Basic Information", status: "Current Status", souls: "Soul Enhancements", ivs: "Individual Potential", active: "Active Skills", learnt: "Learnt Skills", passives: "Passive Skills", work: "Extra Work Suitabilities", disabled: "Disabled Work Preferences", other: "Other Information" },
    fields: { PalID: "Pal Type", Nickname: "Nickname", SkinId: "Skin", Gender: "Gender", Level: "Level", Exp: "Experience", Shiny: "Lucky", PartnerSkillLevel: "Partner Skill Level", CondensedPals: "Condensed Pals", UnusedStatusPoints: "Unused Status Points", FriendshipPoints: "Friendship", IsAwakening: "Awakened", PhysicalHealth: "Physical Health", WorkerSick: "Sickness", ImportedCharacter: "Imported Character", HP: "Health", SP: "Stamina", MP: "Energy", Hunger: "Hunger", MaxHunger: "Maximum Hunger", SAN: "SAN", Support: "Support", CraftSpeed: "Work Speed", Health: "Health", Attack: "Attack", Defense: "Defense", AttackMelee: "Melee Attack", AttackShot: "Ranged Attack", BaseCampBattle: "Base Defense", Anyone: "Any Work" },
    work: { EmitFlame: "Kindling", Watering: "Watering", Seeding: "Planting", GenerateElectricity: "Generating Electricity", Handcraft: "Handiwork", Collection: "Gathering", Deforest: "Lumbering", Mining: "Mining", OilExtraction: "Oil Extraction", ProductMedicine: "Medicine Production", Cool: "Cooling", Transport: "Transporting", MonsterFarm: "Farming" },
    values: { true: "Yes", false: "No", Male: "Male", Female: "Female", None: "None", Healthful: "Healthy", Cold: "Cold", Sprain: "Sprain", Ulcer: "Ulcer", Fracture: "Fracture", Weakness: "Weakness", Depression: "Depression", Bulimia: "Bulimia", Hungry: "Hungry", Starving: "Starving", Injured: "Injured", Sick: "Sick" }, elements: { Normal: "Neutral", Fire: "Fire", Water: "Water", Aqua: "Water", Ice: "Ice", Leaf: "Grass", Earth: "Ground", Electric: "Electric", Electricity: "Electric", Thunder: "Electric", Dark: "Dark", Dragon: "Dragon" }, rank: "Rank", empty: "None",
  },
  ja: {
    sections: { basic: "基本情報", status: "現在の状態", souls: "ソウル強化", ivs: "個体値", active: "アクティブスキル", learnt: "習得済みスキル", passives: "パッシブスキル", work: "追加作業適性", disabled: "無効な作業設定", other: "その他の情報" },
    fields: { PalID: "パル種類", Nickname: "ニックネーム", SkinId: "スキン", Gender: "性別", Level: "レベル", Exp: "経験値", Shiny: "希少", PartnerSkillLevel: "パートナースキルレベル", CondensedPals: "濃縮数", UnusedStatusPoints: "未使用ステータスポイント", FriendshipPoints: "親密度", IsAwakening: "覚醒", PhysicalHealth: "健康状態", WorkerSick: "病気状態", ImportedCharacter: "インポート個体", HP: "体力", SP: "スタミナ", MP: "エネルギー", Hunger: "満腹度", MaxHunger: "最大満腹度", SAN: "SAN値", Support: "サポート", CraftSpeed: "作業速度", Health: "体力", Attack: "攻撃", Defense: "防御", AttackMelee: "近接攻撃", AttackShot: "遠隔攻撃", BaseCampBattle: "拠点戦闘", Anyone: "すべての作業" },
    work: { EmitFlame: "火おこし", Watering: "水やり", Seeding: "種まき", GenerateElectricity: "発電", Handcraft: "手作業", Collection: "採集", Deforest: "伐採", Mining: "採掘", OilExtraction: "採油", ProductMedicine: "製薬", Cool: "冷却", Transport: "運搬", MonsterFarm: "牧場" },
    values: { true: "はい", false: "いいえ", Male: "オス", Female: "メス", None: "なし", Healthful: "健康", Cold: "風邪", Sprain: "捻挫", Ulcer: "胃潰瘍", Fracture: "骨折", Weakness: "衰弱", Depression: "うつ病", Bulimia: "過食症", Hungry: "空腹", Starving: "飢餓", Injured: "負傷", Sick: "病気" }, elements: { Normal: "無", Fire: "炎", Water: "水", Aqua: "水", Ice: "氷", Leaf: "草", Earth: "地", Electric: "雷", Electricity: "雷", Thunder: "雷", Dark: "闇", Dragon: "竜" }, rank: "ランク", empty: "なし",
  },
};

const lang = computed(() => locale.value?.toLowerCase().startsWith("ja") ? "ja" : locale.value?.toLowerCase().startsWith("en") ? "en" : "zh");
const tr = computed(() => text[lang.value]);
const basicKeys = ["PalID", "Nickname", "SkinId", "Gender", "Level", "Exp", "Shiny", "PartnerSkillLevel", "CondensedPals", "UnusedStatusPoints", "FriendshipPoints", "IsAwakening", "ImportedCharacter"];
const statusKeys = ["HP", "SP", "MP", "Hunger", "MaxHunger", "SAN", "Support", "CraftSpeed", "PhysicalHealth", "WorkerSick"];
const reserved = new Set([...basicKeys, ...statusKeys, "PalSouls", "IVs", "ActiveSkills", "LearntSkills", "Passives", "ExtraWorkSuitabilities", "DisableWorkPreferences"]);
const has = (key) => props.palDetail[key] !== undefined && props.palDetail[key] !== null;
const rowsFor = (keys) => keys.filter(has).map((key) => ({ key, label: tr.value.fields[key] || key, value: displayValue(props.palDetail[key]) }));
const objectRows = (key, work = false) => Object.entries(props.palDetail[key] || {}).map(([name, value]) => ({ key: name, label: (work ? tr.value.work[name] : null) || tr.value.fields[name] || name, value: displayValue(value) }));
const otherRows = computed(() => Object.entries(props.palDetail).filter(([key]) => !reserved.has(key)).map(([key, value]) => ({ key, label: tr.value.fields[key] || key, value: displayValue(value) })));

function displayValue(value) {
  if (value === "" || value === null || value === undefined) return tr.value.empty;
  if (value === props.palDetail.PalID) return palName(value);
  if (typeof value === "boolean") return tr.value.values[String(value)];
  if (tr.value.values[value] !== undefined) return tr.value.values[value];
  if (Array.isArray(value)) return value.length ? value.join(", ") : tr.value.empty;
  if (typeof value === "object") return Object.entries(value).map(([key, item]) => `${tr.value.fields[key] || key}: ${displayValue(item)}`).join(" / ");
  return value;
}

const skillDescriptionFallback = {
  Unique_WhiteShieldDragon_ShieldTackle: {
    zh: "艾基鲁迦的专属技能。在前方展开可减轻各种攻击的盾牌，随后向前突进并碾压敌人。",
    en: "Silvegis's exclusive skill. Deploys a shield in front that reduces all types of attacks, then charges forward, crushing the enemy.",
    ja: "シルベージュの専用スキル。正面にあらゆる攻撃を軽減する盾を展開し、前方へ突進して敵を押し潰す。",
  },
};
const noSkillDescription = { zh: "暂无技能说明", en: "No skill description available", ja: "スキル説明はありません" };
const palName = (id) => palMap?.[locale.value]?.[id] || palMap?.[lang.value]?.[id] || palMap?.zh?.[id] || id;
const stripWazaPrefix = (id) => {
  if (typeof id !== "string") return id;
  const idx = id.lastIndexOf(":");
  return idx !== -1 ? id.slice(idx + 1) : id;
};
const skillInfo = (id) => {
  const bareId = stripWazaPrefix(id);
  const meta = activeSkillById.get(bareId) || {};
  const localized = skillMap?.[locale.value]?.[bareId] || skillMap?.[lang.value]?.[bareId] || {};
  const langKey = lang.value === "zh" ? "zh" : lang.value;
  const localizedName = localized.name || meta[langKey] || meta.name;
  return {
    name: localizedName || bareId,
    description: localized.desc || skillDescriptionFallback[bareId]?.[lang.value] || noSkillDescription[lang.value],
    element: meta.element || "Normal",
  };
};
const passiveInfo = (id) => {
  const meta = passiveById.get(id) || {};
  const localized = passiveI18n?.[lang.value]?.[id] || {};
  return { name: localized.name || meta[lang.value === "zh" ? "zh" : lang.value] || meta.name || id, description: localized.desc || id, rank: Number(meta.rank || 0) };
};
const elementColor = (element) => elementColors[element] || elementColors.Normal;
const passiveColor = (rank) => passiveRankColors[rank] || passiveRankColors[0];
const list = (key) => Array.isArray(props.palDetail[key]) ? props.palDetail[key] : [];
</script>

<template>
  <div class="pal-detail-view">
    <section v-if="rowsFor(basicKeys).length" class="detail-section">
      <div class="section-title">{{ tr.sections.basic }}</div>
      <div class="info-list"><div v-for="row in rowsFor(basicKeys)" :key="row.key" class="info-row"><span>{{ row.label }}</span><strong>{{ row.value }}</strong></div></div>
    </section>
    <section v-if="rowsFor(statusKeys).length" class="detail-section">
      <div class="section-title">{{ tr.sections.status }}</div>
      <div class="info-list"><div v-for="row in rowsFor(statusKeys)" :key="row.key" class="info-row"><span>{{ row.label }}</span><strong>{{ row.value }}</strong></div></div>
    </section>
    <section v-if="objectRows('PalSouls').length" class="detail-section">
      <div class="section-title">{{ tr.sections.souls }}</div>
      <div class="info-list"><div v-for="row in objectRows('PalSouls')" :key="row.key" class="info-row"><span>{{ row.label }}</span><strong>{{ row.value }}</strong></div></div>
    </section>
    <section v-if="objectRows('IVs').length" class="detail-section">
      <div class="section-title">{{ tr.sections.ivs }}</div>
      <div class="info-list"><div v-for="row in objectRows('IVs')" :key="row.key" class="info-row"><span>{{ row.label }}</span><strong>{{ row.value }}</strong></div></div>
    </section>
    <section v-if="objectRows('ExtraWorkSuitabilities', true).length" class="detail-section full-width">
      <div class="section-title">{{ tr.sections.work }}</div>
      <div class="info-list"><div v-for="row in objectRows('ExtraWorkSuitabilities', true)" :key="row.key" class="info-row"><span>{{ row.label }}</span><strong>Lv.{{ row.value }}</strong></div></div>
    </section>
    <section v-for="group in [{ key: 'ActiveSkills', title: tr.sections.active }, { key: 'LearntSkills', title: tr.sections.learnt }]" v-show="list(group.key).length" :key="group.key" class="detail-section full-width">
      <div class="section-title">{{ group.title }}</div>
      <div class="tag-list">
        <n-tooltip v-for="skill in list(group.key)" :key="skill" trigger="hover">
          <template #trigger>
            <n-tag size="small" :color="{ color: `${elementColor(skillInfo(skill).element)}22`, textColor: elementColor(skillInfo(skill).element), borderColor: `${elementColor(skillInfo(skill).element)}88` }">
              <span class="element-icon" :style="{ backgroundColor: elementColor(skillInfo(skill).element) }">{{ elementIcons[skillInfo(skill).element] || '○' }}</span>
              {{ skillInfo(skill).name }}
            </n-tag>
          </template>
          <div class="skill-tooltip"><strong>{{ skillInfo(skill).name }}</strong><div>{{ tr.elements[skillInfo(skill).element] || skillInfo(skill).element }}</div><div>{{ skillInfo(skill).description }}</div></div>
        </n-tooltip>
      </div>
    </section>
    <section v-if="list('Passives').length" class="detail-section full-width">
      <div class="section-title">{{ tr.sections.passives }}</div>
      <div class="tag-list">
        <n-tooltip v-for="passive in list('Passives')" :key="passive" trigger="hover">
          <template #trigger>
            <n-tag size="small" :color="{ color: `${passiveColor(passiveInfo(passive).rank)}22`, textColor: passiveColor(passiveInfo(passive).rank), borderColor: `${passiveColor(passiveInfo(passive).rank)}88` }">
              {{ passiveInfo(passive).name }} · {{ tr.rank }} {{ passiveInfo(passive).rank }}
            </n-tag>
          </template>
          <div class="skill-tooltip"><strong>{{ passiveInfo(passive).name }}</strong><div>{{ tr.rank }} {{ passiveInfo(passive).rank }}</div><div>{{ passiveInfo(passive).description }}</div></div>
        </n-tooltip>
      </div>
    </section>
    <section v-if="list('DisableWorkPreferences').length" class="detail-section full-width">
      <div class="section-title">{{ tr.sections.disabled }}</div>
      <div class="tag-list"><n-tag v-for="item in list('DisableWorkPreferences')" :key="item" size="small" type="error">{{ tr.fields[item] || item }}</n-tag></div>
    </section>
    <section v-if="otherRows.length" class="detail-section full-width">
      <div class="section-title">{{ tr.sections.other }}</div>
      <div class="info-list"><div v-for="row in otherRows" :key="row.key" class="info-row"><span>{{ row.label }}</span><strong>{{ row.value }}</strong></div></div>
    </section>
  </div>
</template>

<style scoped>
.pal-detail-view { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.detail-section { min-width: 0; }
.full-width { grid-column: 1 / -1; }
.section-title { margin-bottom: 8px; font-size: 15px; font-weight: 600; }
.info-list { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1px 16px; }
.info-row { display: flex; justify-content: space-between; gap: 12px; padding: 7px 0; border-bottom: 1px solid rgba(128, 128, 128, 0.16); }
.info-row span { color: #888; }
.info-row strong { text-align: right; overflow-wrap: anywhere; }
.tag-list { display: flex; flex-wrap: wrap; gap: 8px; }
.element-icon { display: inline-flex; align-items: center; justify-content: center; min-width: 20px; height: 16px; margin-right: 5px; padding: 0 3px; border-radius: 3px; color: #fff; font-size: 10px; line-height: 1; }
.skill-tooltip { max-width: 360px; line-height: 1.55; }
.skill-tooltip strong { display: block; margin-bottom: 4px; }
@media (max-width: 700px) { .pal-detail-view, .info-list { grid-template-columns: 1fr; } }
</style>
