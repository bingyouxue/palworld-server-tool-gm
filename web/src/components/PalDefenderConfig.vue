<script setup>
import { ref, watch } from "vue";
import { useMessage } from "naive-ui";
const TOKEN = "palworld_token";

const props = defineProps({ show: { type: Boolean, default: false } });
const message = useMessage();

const activeTab = ref("config");
const loading   = ref(false);
const saving    = ref(false);

// Config.json state
const configExists = ref(false);
const cfg = ref({
  EnableRestAPI: false, RestAPIPort: 8080,
  MOTD: [],
  shouldWarnCheaters: true, shouldWarnCheatersReason: true,
  shouldKickCheaters: false, shouldBanCheaters: false, shouldIPBanCheaters: false,
  steamidProtection: true, blockTowerBossCapture: true,
  disableIllegalItemProtection: false, doActionUponIllegalPalStats: true,
  palStatsMaxRank: -1, treeLimiter: 0,
  pvpMaxToBuildingDamage: 0, pvpMaxToPlayerDamage: 0, pvpMaxToPalDamage: 0,
  pveMaxToPalBanThreshold: 0,
  useWhitelist: false, useAdminWhitelist: false, adminAutoLogin: false,
  preventAdminPasswordInChat: true, allowAdminCheats: false, allowGodmodeOnehit: false,
  chatBypassWait: false, chatMessageMaxLen: 256,
  announceConnections: true, dontAnnounceAdminConnections: false,
  announcePunishments: true, announcePlayerDeaths: false,
  announceOpenOilrigBoxes: false, announceHelicopterKills: false,
  announcePlayerSummons: false, announceAdminSummons: false,
  logChat: true, logRCON: true, logPlayerLogins: true, logPlayerDeaths: false,
  logPlayerBuildings: false, logPlayerSummons: false, logPlayerCaptures: false,
  logCraftings: false, logTechUnlocks: false, logPlayerUID: true, logPlayerIP: false, logNetworking: false,
  exitServerOnStartupFailure: false, disableButchering: false, disableRenaming: false,
  disablePalRenaming: false, RCONUsePacketIdFix: true,
  OilrigGoalBoxLocktime: 300, RCONTimeout: 10,
});
const cfgRaw     = ref("");
const cfgRawMode = ref(false);

// Default.json state
const defExists  = ref(false);
const def = ref({
  PalSelectionMode: "AllowAllExceptBanned",
  AllowedPalIDs: [], BannedPalIDs: [],
  MaxValueLimitAction: "BlockImport",
  DisallowedPassivesAction: "BlockImport",
  DisallowedPassives: [],
  Disabled: false, BanIfPalIsImpossible: false, AllowGenderNone: false,
  MaxLevel: 65, MaxRank: 5,
  PalSouls: { Health: 20, Attack: 20, Defense: 20, CraftSpeed: 20 },
  IVs: { Health: 100, AttackMelee: 100, AttackShot: 100, Defense: 100 },
});
const defRaw     = ref("");
const defRawMode = ref(false);

// ExampleOverride.json state
const exExists = ref(false);
const exRaw    = ref("");

const authHeader = () => ({
  "Content-Type": "application/json",
  Authorization: `Bearer ${localStorage.getItem(TOKEN) || ""}`,
});

async function loadTab(tab) {
  loading.value = true;
  try {
    if (tab === "config") {
      const r = await fetch("/api/paldefender/config", { headers: authHeader() });
      const j = await r.json();
      configExists.value = !!j.exists;
      if (j.exists && j.data) {
        Object.assign(cfg.value, j.data);
        cfgRaw.value = JSON.stringify(j.data, null, 2);
      }
    } else if (tab === "default") {
      const r = await fetch("/api/paldefender/import-rules/default", { headers: authHeader() });
      const j = await r.json();
      defExists.value = !!j.exists;
      if (j.exists && j.data) {
        Object.assign(def.value, j.data);
        if (j.data.PalSouls) Object.assign(def.value.PalSouls, j.data.PalSouls);
        if (j.data.IVs) Object.assign(def.value.IVs, j.data.IVs);
        defRaw.value = JSON.stringify(j.data, null, 2);
      }
    } else {
      const r = await fetch("/api/paldefender/import-rules/example", { headers: authHeader() });
      const j = await r.json();
      exExists.value = !!j.exists;
      if (j.exists && j.data) exRaw.value = JSON.stringify(j.data, null, 2);
    }
  } finally { loading.value = false; }
}

async function saveTab(tab) {
  saving.value = true;
  try {
    let body, url;
    if (tab === "config") {
      body = cfgRawMode.value ? cfgRaw.value : JSON.stringify(cfg.value);
      url  = "/api/paldefender/config";
    } else if (tab === "default") {
      body = defRawMode.value ? defRaw.value : JSON.stringify(def.value);
      url  = "/api/paldefender/import-rules/default";
    } else {
      body = exRaw.value;
      url  = "/api/paldefender/import-rules/example";
    }
    const r = await fetch(url, { method: "PUT", headers: authHeader(), body });
    const j = await r.json();
    if (!r.ok) { message.error(j.error || "保存失败"); return; }
    message.success("已保存");
    await loadTab(tab);
  } finally { saving.value = false; }
}

watch(() => props.show, v => { if (v) loadTab(activeTab.value); });
watch(activeTab, v => { if (props.show) loadTab(v); });
</script>

<template>
  <div v-if="show">
    <n-tabs v-model:value="activeTab" type="line" animated>

      <!-- Config.json -->
      <n-tab-pane name="config" tab="Config.json">
        <n-spin :show="loading">
          <n-alert v-if="!configExists" type="info" class="mb-3">
            Config.json 尚不存在（服务器首次启动后自动生成），以下为默认值，可提前编辑。
          </n-alert>
          <n-scrollbar v-if="!cfgRawMode" style="max-height:52vh">
            <div class="pd-pane">
              <n-card size="small" class="mb-2" title="REST API">
                <n-grid :cols="2" :x-gap="12" :y-gap="4">
                  <n-gi>
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>启用 REST API (EnableRestAPI)</span>
                      <n-switch v-model:value="cfg.EnableRestAPI" size="small"/>
                    </n-flex>
                  </n-gi>
                  <n-gi>
                    <n-form-item label="端口 (RestAPIPort)" label-placement="left" label-width="165" size="small">
                      <n-input-number v-model:value="cfg.RestAPIPort" :min="1024" :max="65535" size="small" style="width:100%"/>
                    </n-form-item>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="登入公告 (MOTD)">
                <n-input
                  :value="Array.isArray(cfg.MOTD)?cfg.MOTD.join('\n'):(cfg.MOTD||'')"
                  @update:value="v => cfg.MOTD = v.split('\n')"
                  type="textarea" :rows="3" size="small"
                  placeholder="Welcome {PlayerName}!"/>
              </n-card>

              <n-card size="small" class="mb-2" title="反外挂处置">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'shouldWarnCheaters',l:'警告作弊者'},
                    {k:'shouldWarnCheatersReason',l:'警告时附上原因'},
                    {k:'shouldKickCheaters',l:'自动踢出作弊者'},
                    {k:'shouldBanCheaters',l:'自动封禁作弊者'},
                    {k:'shouldIPBanCheaters',l:'自动 IP 封禁作弊者'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="cfg[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="漏洞防护">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'steamidProtection',l:'防止重复 UserId 登入'},
                    {k:'blockTowerBossCapture',l:'禁止捕捉塔主'},
                    {k:'disableIllegalItemProtection',l:'停用非法道具防护'},
                    {k:'doActionUponIllegalPalStats',l:'自动处理异常帕鲁数值'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="cfg[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
                <n-grid :cols="2" :x-gap="12" :y-gap="4" class="mt-2">
                  <n-gi v-for="nf in [
                    {k:'palStatsMaxRank',l:'帕鲁强化上限 (-1=自动)',min:-1,max:20},
                    {k:'treeLimiter',l:'砍树速率(s/棵，0=关)',min:0,max:5,step:0.05},
                    {k:'pvpMaxToBuildingDamage',l:'PvP 建筑最大伤害',min:0,max:100000},
                    {k:'pvpMaxToPlayerDamage',l:'PvP 玩家最大伤害',min:0,max:100000},
                    {k:'pvpMaxToPalDamage',l:'PvP 帕鲁最大伤害',min:0,max:100000},
                    {k:'pveMaxToPalBanThreshold',l:'PvE 帕鲁伤害封禁阈值',min:0,max:1000000},
                  ]" :key="nf.k">
                    <div class="num-row">
                      <span class="num-label">{{ nf.l }}</span>
                      <n-input-number v-model:value="cfg[nf.k]" :min="nf.min" :max="nf.max" :step="nf.step||1" size="small" style="width:110px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="白名单与管理员">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'useWhitelist',l:'启用白名单'},
                    {k:'useAdminWhitelist',l:'启用管理员 IP 白名单'},
                    {k:'adminAutoLogin',l:'白名单管理员自动管理模式'},
                    {k:'preventAdminPasswordInChat',l:'防止密码在聊天外泄'},
                    {k:'allowAdminCheats',l:'允许管理员作弊指令'},
                    {k:'allowGodmodeOnehit',l:'godmode 一击击杀'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="cfg[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="公告 / 聊天">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'announceConnections',l:'公告玩家上下线'},
                    {k:'dontAnnounceAdminConnections',l:'不公告管理员上下线'},
                    {k:'announcePunishments',l:'公告作弊处罚'},
                    {k:'announcePlayerDeaths',l:'公告玩家死亡'},
                    {k:'announceOpenOilrigBoxes',l:'公告钻油台宝箱开启'},
                    {k:'announceHelicopterKills',l:'公告直升机击杀'},
                    {k:'announcePlayerSummons',l:'公告玩家召唤帕鲁'},
                    {k:'announceAdminSummons',l:'公告管理员召唤帕鲁'},
                    {k:'chatBypassWait',l:'移除聊天冷却时间'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="cfg[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
                <div class="num-row mt-2">
                  <span class="num-label">聊天消息最大长度 (chatMessageMaxLen)</span>
                  <n-input-number v-model:value="cfg.chatMessageMaxLen" :min="1" :max="1000" size="small" style="width:110px"/>
                </div>
              </n-card>

              <n-card size="small" class="mb-2" title="日志">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'logChat',l:'记录聊天消息'},
                    {k:'logRCON',l:'记录 RCON 指令'},
                    {k:'logPlayerLogins',l:'记录玩家上下线'},
                    {k:'logPlayerDeaths',l:'记录玩家死亡'},
                    {k:'logPlayerBuildings',l:'记录玩家建筑操作'},
                    {k:'logPlayerSummons',l:'记录玩家召唤帕鲁'},
                    {k:'logPlayerCaptures',l:'记录玩家捕捉帕鲁'},
                    {k:'logCraftings',l:'记录玩家制作'},
                    {k:'logTechUnlocks',l:'记录科技解锁'},
                    {k:'logPlayerUID',l:'日志记录玩家 UID'},
                    {k:'logPlayerIP',l:'日志记录玩家 IP'},
                    {k:'logNetworking',l:'记录客户端网络数据'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="cfg[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="其他">
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'exitServerOnStartupFailure',l:'启动失败时关闭服务器'},
                    {k:'disableButchering',l:'停用屠宰'},
                    {k:'disableRenaming',l:'停用角色改名'},
                    {k:'disablePalRenaming',l:'停用帕鲁改名'},
                    {k:'RCONUsePacketIdFix',l:'修正 RCON 封包 ID'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="cfg[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
                <n-grid :cols="2" :x-gap="12" :y-gap="4" class="mt-2">
                  <n-gi>
                    <div class="num-row">
                      <span class="num-label">钻油台宝箱锁定(s)</span>
                      <n-input-number v-model:value="cfg.OilrigGoalBoxLocktime" :min="0" :max="3600" size="small" style="width:110px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="num-row">
                      <span class="num-label">RCON 连接超时(s)</span>
                      <n-input-number v-model:value="cfg.RCONTimeout" :min="1" :max="60" :step="0.5" size="small" style="width:110px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>
            </div>
          </n-scrollbar>
          <n-input v-else v-model:value="cfgRaw" type="textarea" :rows="22" style="font-family:monospace;font-size:12px"/>
        </n-spin>
        <n-flex class="mt-3" justify="end" :size="8">
          <n-button size="small" @click="cfgRawMode=!cfgRawMode">{{ cfgRawMode ? '可视化编辑' : '编辑原始文件' }}</n-button>
          <n-button size="small" @click="loadTab('config')" :loading="loading">刷新</n-button>
          <n-button size="small" type="primary" @click="saveTab('config')" :loading="saving">保存</n-button>
        </n-flex>
      </n-tab-pane>

      <!-- Default.json -->
      <n-tab-pane name="default" tab="Default.json">
        <n-spin :show="loading">
          <n-alert v-if="!defExists" type="info" class="mb-3">Default.json 不存在，以下为默认值。</n-alert>
          <n-scrollbar v-if="!defRawMode" style="max-height:52vh">
            <div class="pd-pane">
              <n-card size="small" class="mb-2" title="帕鲁筛选模式">
                <n-form-item label="PalSelectionMode" label-placement="left" label-width="160" size="small">
                  <n-select v-model:value="def.PalSelectionMode" size="small" :options="[
                    {label:'AllowAllExceptBanned — 允许所有（黑名单除外）',value:'AllowAllExceptBanned'},
                    {label:'AllowOnlyListed — 只允许白名单帕鲁',value:'AllowOnlyListed'},
                  ]"/>
                </n-form-item>
                <n-grid :cols="2" :x-gap="12" :y-gap="2">
                  <n-gi v-for="sw in [
                    {k:'Disabled',l:'停用此规则文件'},
                    {k:'BanIfPalIsImpossible',l:'封禁不可能存在的帕鲁'},
                    {k:'AllowGenderNone',l:'允许无性别帕鲁'},
                  ]" :key="sw.k">
                    <n-flex align="center" justify="space-between" class="tog">
                      <span>{{ sw.l }}</span>
                      <n-switch v-model:value="def[sw.k]" size="small"/>
                    </n-flex>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="数值上限">
                <n-grid :cols="2" :x-gap="12" :y-gap="6">
                  <n-gi>
                    <div class="num-row"><span class="num-label">MaxLevel</span>
                      <n-input-number v-model:value="def.MaxLevel" :min="1" :max="65" size="small" style="width:90px"/>
                    </div>
                  </n-gi>
                  <n-gi>
                    <div class="num-row"><span class="num-label">MaxRank</span>
                      <n-input-number v-model:value="def.MaxRank" :min="0" :max="5" size="small" style="width:90px"/>
                    </div>
                  </n-gi>
                </n-grid>
                <n-form-item label="MaxValueLimitAction" label-placement="left" label-width="175" size="small" class="mt-2">
                  <n-select v-model:value="def.MaxValueLimitAction" size="small" :options="[
                    {label:'BlockImport — 超限时拒绝导入',value:'BlockImport'},
                    {label:'ClampToMaxValues — 超限时截断至上限',value:'ClampToMaxValues'},
                  ]"/>
                </n-form-item>
                <n-text depth="3" style="font-size:11px;display:block;margin:6px 0 4px">PalSouls 每属性上限</n-text>
                <n-grid :cols="4" :x-gap="8" :y-gap="4">
                  <n-gi v-for="k in ['Health','Attack','Defense','CraftSpeed']" :key="k">
                    <div class="num-col">
                      <span class="num-label">{{ k }}</span>
                      <n-input-number v-model:value="def.PalSouls[k]" :min="0" :max="100" size="small" style="width:75px"/>
                    </div>
                  </n-gi>
                </n-grid>
                <n-text depth="3" style="font-size:11px;display:block;margin:6px 0 4px">IVs 每属性上限</n-text>
                <n-grid :cols="4" :x-gap="8" :y-gap="4">
                  <n-gi v-for="k in ['Health','AttackMelee','AttackShot','Defense']" :key="k">
                    <div class="num-col">
                      <span class="num-label">{{ k }}</span>
                      <n-input-number v-model:value="def.IVs[k]" :min="0" :max="100" size="small" style="width:75px"/>
                    </div>
                  </n-gi>
                </n-grid>
              </n-card>

              <n-card size="small" class="mb-2" title="违禁被动技能">
                <n-form-item label="DisallowedPassivesAction" label-placement="left" label-width="200" size="small">
                  <n-select v-model:value="def.DisallowedPassivesAction" size="small" :options="[
                    {label:'BlockImport — 拒绝导入',value:'BlockImport'},
                    {label:'RemoveFromPal — 导入前移除',value:'RemoveFromPal'},
                  ]"/>
                </n-form-item>
                <n-dynamic-tags v-model:value="def.DisallowedPassives"/>
              </n-card>

              <n-card size="small" class="mb-2" title="封禁帕鲁 ID (BannedPalIDs)">
                <n-dynamic-tags v-model:value="def.BannedPalIDs"/>
              </n-card>

              <n-card size="small" class="mb-2" title="允许帕鲁 ID (AllowedPalIDs — AllowOnlyListed 模式下有效)">
                <n-dynamic-tags v-model:value="def.AllowedPalIDs"/>
              </n-card>
            </div>
          </n-scrollbar>
          <n-input v-else v-model:value="defRaw" type="textarea" :rows="22" style="font-family:monospace;font-size:12px"/>
        </n-spin>
        <n-flex class="mt-3" justify="end" :size="8">
          <n-button size="small" @click="defRawMode=!defRawMode">{{ defRawMode ? '可视化编辑' : '编辑原始文件' }}</n-button>
          <n-button size="small" @click="loadTab('default')" :loading="loading">刷新</n-button>
          <n-button size="small" type="primary" @click="saveTab('default')" :loading="saving">保存</n-button>
        </n-flex>
      </n-tab-pane>

      <!-- ExampleOverride.json -->
      <n-tab-pane name="example" tab="ExampleOverride.json">
        <n-spin :show="loading">
          <n-alert v-if="!exExists" type="info" class="mb-3">ExampleOverride.json 不存在（服务器首次启动后自动生成）。</n-alert>
          <n-alert type="default" class="mb-2" style="font-size:11px">
            此文件为特定帕鲁 ID 规则覆盖的示例模板，直接编辑 JSON 保存即可。
          </n-alert>
          <n-input v-model:value="exRaw" type="textarea" :rows="20" style="font-family:monospace;font-size:12px"/>
        </n-spin>
        <n-flex class="mt-3" justify="end" :size="8">
          <n-button size="small" @click="loadTab('example')" :loading="loading">刷新</n-button>
          <n-button size="small" type="primary" @click="saveTab('example')" :loading="saving">保存</n-button>
        </n-flex>
      </n-tab-pane>

    </n-tabs>
  </div>
</template>

<style scoped>
.pd-pane { padding-right: 4px; }
.tog { padding: 3px 0; }
.num-row { display: flex; align-items: center; justify-content: space-between; padding: 2px 0; gap: 6px; }
.num-col { display: flex; flex-direction: column; gap: 3px; }
.num-label { font-size: 12px; opacity: .8; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mb-2 { margin-bottom: 8px; }
.mb-3 { margin-bottom: 12px; }
.mt-2 { margin-top: 8px; }
.mt-3 { margin-top: 12px; }
</style>
