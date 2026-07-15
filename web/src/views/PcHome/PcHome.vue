<script setup>
import { computed, h, inject, onMounted, ref, watch } from "vue";
import {
  AdminPanelSettingsOutlined,
  SupervisedUserCircleRound,
  SettingsPowerRound,
  ArchiveOutlined,
  PublicRound,
  DashboardOutlined,
} from "@vicons/material";
import {
  GameController,
  LanguageSharp,
  ShieldCheckmarkSharp,
  Terminal,
  Settings,
} from "@vicons/ionicons5";
import { GuiManagement } from "@vicons/carbon";
import { BroadcastTower } from "@vicons/fa";
import { NIcon, useMessage } from "naive-ui";
import { useI18n } from "vue-i18n";
import ApiService from "@/service/api";
import pageStore from "@/stores/model/page.js";
import PlayerList from "./component/PlayerList.vue";
import GuildList from "./component/GuildList.vue";
import MapView from "./component/MapView.vue";
import ItemManager from "./component/ItemManager.vue";
import GameConfigModal from "./component/GameConfigModal.vue";
import RconManager from "@/components/RconManager.vue";
import AdminOverview from "@/components/AdminOverview.vue";
import BackupManager from "@/components/BackupManager.vue";
import BroadcastComposer from "@/components/BroadcastComposer.vue";
import ShutdownDialog from "@/components/ShutdownDialog.vue";
import WhitelistManager from "@/components/WhitelistManager.vue";
import SetupWizard from "@/components/SetupWizard.vue";
import whitelistStore from "@/stores/model/whitelist";
import playerToGuildStore from "@/stores/model/playerToGuild";
import userStore from "@/stores/model/user";

const emit = defineEmits(["open-config"]);

const { t, locale } = useI18n();

const message = useMessage();
const PALWORLD_TOKEN = "palworld_token";

const pageWidth = computed(() => pageStore().getScreenWidth());
const smallScreen = computed(() => pageWidth.value < 1024);

const loading = ref(false);
const serverInfo = ref({});
const serverMetrics = ref({});
const currentDisplay = ref("players");
const playerList = ref([]);
const onlinePlayerList = ref([]);
const guildList = ref([]);
const languageOptions = ref([]);
const asArray = (value) => (Array.isArray(value) ? value : []);

const isLogin = ref(false);

// Use shared dark mode from App.vue (localStorage-backed, manually toggleable)
const isDarkMode = inject("isDarkMode", ref(
  window.matchMedia("(prefers-color-scheme: dark)").matches,
));
const toggleDarkMode = inject("toggleDarkMode", () => {
  isDarkMode.value = !isDarkMode.value;
});

const updateDarkMode = (e) => {
  if (localStorage.getItem("pst_dark_mode") === null) {
    isDarkMode.value = e.matches;
  }
};

const handleSelectLanguage = (key) => {
  message.info(t("message.changelanguage"));
  if (key === "zh") {
    localStorage.setItem("locale", "zh");
    // locale.value = "zh";
  } else if (key === "ja") {
    localStorage.setItem("locale", "ja");
    // locale.value = "ja";
  } else {
    localStorage.setItem("locale", "en");
    // locale.value = "en";
  }
  setTimeout(() => {
    location.reload();
  }, 1000);
};

const toPalConf = () => {
  window.open(
    "https://pal-conf.bluefissure.com/",
    "_blank",
    "noopener,noreferrer",
  );
};

const toGithub = () => {
  window.open("https://github.com/zaigie/palworld-server-tool/releases");
};
const serverToolInfo = ref({});
const hasNewVersion = ref(false);
const getServerToolInfo = async () => {
  const { data } = await new ApiService().getServerToolInfo();
  serverToolInfo.value = data.value;
  if (data.value) {
    hasNewVersion.value = isNewVersion(data.value?.version, data.value?.latest);
  }
};
const isNewVersion = (version, latest) => {
  if (
    typeof version !== "string" ||
    typeof latest !== "string" ||
    version === "Unknown" ||
    version === "Develop" ||
    latest === ""
  ) {
    return false;
  }
  const currentParts = version.replace(/^v/i, "").split(".").map(Number);
  const latestParts = latest.replace(/^v/i, "").split(".").map(Number);
  if (currentParts.some(Number.isNaN) || latestParts.some(Number.isNaN))
    return false;
  const partCount = Math.max(currentParts.length, latestParts.length);
  for (let i = 0; i < partCount; i++) {
    const currentPart = currentParts[i] || 0;
    const latestPart = latestParts[i] || 0;
    if (latestPart > currentPart) {
      return true;
    } else if (latestPart < currentPart) {
      return false;
    }
  }
  return false;
};

// get data
const getServerInfo = async () => {
  const { data } = await new ApiService().getServerInfo();
  serverInfo.value = data.value || {};
};

const getServerMetrics = async () => {
  const { data } = await new ApiService().getServerMetrics();
  serverMetrics.value = data.value;
};

const getPlayerList = async () => {
  getOnlineList();
  const { data } = await new ApiService().getPlayerList({
    order_by: "last_online",
    desc: true,
  });
  playerList.value = asArray(data.value);
};
const getOnlineList = async () => {
  const { data } = await new ApiService().getOnlinePlayerList();
  onlinePlayerList.value = asArray(data.value);
};

// login
const showLoginModal = ref(false);
const password = ref("");
const handleLogin = async () => {
  const { data, statusCode } = await new ApiService().login({
    password: password.value,
  });
  if (statusCode.value === 401) {
    message.error(t("message.autherr"));
    password.value = "";
    return;
  }
  let token = data.value.token;
  localStorage.setItem(PALWORLD_TOKEN, token);
  userStore().setIsLogin(true, token);
  await getWhiteList();
  message.success(t("message.authsuccess"));
  showLoginModal.value = false;
  isLogin.value = true;
  currentDisplay.value = "overview";
};
const showRconDrawer = ref(false);
const handleRconDrawer = () => {
  if (checkAuthToken()) {
    showRconDrawer.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};

// 控制中心（下拉菜单）
// 包含：白名单管理、RCON 命令、游戏内广播、关闭服务器
const renderIcon = (icon, color = "#666") => {
  return () => {
    return h(
      NIcon,
      {
        color: color,
      },
      {
        default: () => h(icon),
      },
    );
  };
};
const controlCenterOption = [
  {
    label: () => t("configuration.title"),
    key: "settings",
    icon: renderIcon(Settings),
  },
  // {
  //   label: () => {
  //     return h("div", null, {
  //       default: () => t("button.backup"),
  //     });
  //   },
  //   key: "backup",
  //   icon: renderIcon(ArchiveOutlined),
  // },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.palconf"),
      });
    },
    key: "palconf",
    icon: renderIcon(Settings),
  },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.whitelist"),
      });
    },
    key: "whitelist",
    icon: renderIcon(ShieldCheckmarkSharp),
  },
  // {
  //   label: () => {
  //     return h("div", null, {
  //       default: () => t("button.rcon"),
  //     });
  //   },
  //   key: "rcon",
  //   icon: renderIcon(Terminal),
  // },
  {
    label: () => {
      return h("div", null, {
        default: () => t("button.broadcast"),
      });
    },
    key: "broadcast",
    icon: renderIcon(BroadcastTower),
  },
  {
    label: () => {
      return h(
        "div",
        {
          style: { color: "#cc2d48" },
        },
        {
          default: () => t("button.shutdown"),
        },
      );
    },
    key: "shutdown",
    icon: renderIcon(SettingsPowerRound, "#cc2d48"),
  },
];
const handleSelectControlCenter = (key) => {
  if (key === "settings") {
    if (checkAuthToken()) emit("open-config");
    else {
      message.error(t("message.requireauth"));
      showLoginModal.value = true;
    }
  } else if (key === "palconf") {
    toPalConf();
  } else if (key === "whitelist") {
    handleWhiteList();
  } else if (key === "rcon") {
    handleRconDrawer();
  } else if (key === "broadcast") {
    handleStartBrodcast();
  } else if (key === "shutdown") {
    handleShutdown();
  } else {
    message.error("错误");
  }
};

// 白名单
const showWhiteListModal = ref(false);
const handleWhiteList = () => {
  if (checkAuthToken()) {
    showWhiteListModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};
const getWhiteList = async () => {
  if (checkAuthToken()) {
    const { data, statusCode } = await new ApiService().getWhitelist();
    if (statusCode.value === 200) {
      if (data.value) {
        whitelistStore().setWhitelist(asArray(data.value));
      }
    }
  }
};
// 接受玩家加入到黑名单信息
const getSonWhitelistStatus = () => {
  getWhiteList();
};

// 广播
const showBroadcastModal = ref(false);
const handleStartBrodcast = () => {
  // 开始广播
  if (checkAuthToken()) {
    showBroadcastModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};
const showShutdownDialog = ref(false);
const handleShutdown = () => {
  if (checkAuthToken()) {
    showShutdownDialog.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};

const showGameConfigModal = ref(false);
const handleGameConfig = () => {
  if (checkAuthToken()) {
    showGameConfigModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};
const toPlayers = async () => {
  if (currentDisplay.value === "players") {
    return;
  }
  currentDisplay.value = "players";
  playerToGuildStore().setUpdateStatus("players");
};
const toOverview = () => {
  currentDisplay.value = "overview";
};
const toGuilds = async () => {
  if (currentDisplay.value === "guilds") {
    return;
  }
  currentDisplay.value = "guilds";
  playerToGuildStore().setUpdateStatus("guilds");
};

const toMap = async () => {
  if (currentDisplay.value === "map") {
    return;
  }
  currentDisplay.value = "map";
  playerToGuildStore().setUpdateStatus("map");
};

const toItems = () => {
  if (!checkAuthToken()) {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
    return;
  }
  currentDisplay.value = "items";
};

const playerToGuildStatus = computed(() =>
  playerToGuildStore().getUpdateStatus(),
);

watch(
  () => playerToGuildStatus.value,
  (newVal) => {
    currentDisplay.value = newVal;
    if (newVal === "players") {
    } else if (newVal === "guilds") {
    }
  },
);

/**
 * 检测 token
 */
const checkAuthToken = () => {
  const token = localStorage.getItem(PALWORLD_TOKEN);
  if (token && token !== "") {
    if (isTokenExpired(token)) {
      localStorage.removeItem(PALWORLD_TOKEN);
      return false;
    }
    isLogin.value = true;
    return true;
  }
  return false;
};
const isTokenExpired = (token) => {
  const payload = JSON.parse(atob(token.split(".")[1]));
  return payload.exp < Date.now() / 1000;
};

const backupModal = ref(false);
const handleBackupList = () => {
  if (checkAuthToken()) {
    backupModal.value = true;
  } else {
    message.error(t("message.requireauth"));
    showLoginModal.value = true;
  }
};

onMounted(async () => {
  locale.value = localStorage.getItem("locale");
  languageOptions.value = [
    {
      label: "简体中文",
      key: "zh",
      disabled: locale.value == "zh",
    },
    {
      label: "English",
      key: "en",
      disabled: locale.value == "en",
    },
    {
      label: "日本語",
      key: "ja",
      disabled: locale.value == "ja",
    },
  ];
  loading.value = true;
  checkAuthToken();
  await Promise.all([
    getServerInfo(),
    getServerMetrics(),
    getServerToolInfo(),
    getPlayerList(),
  ]);
  await getWhiteList();
  if (isLogin.value) currentDisplay.value = "overview";
  loading.value = false;
  setInterval(async () => {
    await getPlayerList();
    await getServerMetrics();
  }, 10000);
  // 调试用
  // currentDisplay.value = "map";
  // playerToGuildStore().setUpdateStatus("map");
});
</script>

<template>
  <div class="home-page">
    <div
      :class="isDarkMode ? 'bg-#18181c text-#fff' : 'bg-#fff text-#18181c'"
      class="flex justify-between items-center p-3"
    >
      <n-space class="flex items-center">
        <span
          class="line-clamp-1"
          :class="smallScreen ? 'text-lg' : 'text-2xl'"
          >{{ $t("title") }}</span
        >
        <n-badge
          v-if="serverToolInfo?.version"
          :value="hasNewVersion ? 'new' : ''"
        >
          <n-tag
            type="warning"
            :size="smallScreen ? 'mini' : 'medium'"
            round
            @click="toGithub"
            style="cursor: pointer"
            >{{ serverToolInfo.version }}</n-tag
          >
        </n-badge>
        <n-tooltip trigger="hover">
          <template #trigger>
            <n-tag type="default" :size="smallScreen ? 'medium' : 'large'">{{
              serverInfo?.name
                ? `${serverInfo.name + " " + serverInfo.version}`
                : $t("status.serverUnavailable")
            }}</n-tag>
          </template>
          <div>
            <p>{{ $t("item.serverFps") }}: {{ serverMetrics?.server_fps }}</p>
            <p>{{ $t("item.serverUptime") }}: {{ serverMetrics?.uptime }}(s)</p>
            <p>{{ $t("item.serverDays") }}: {{ serverMetrics?.days }}</p>
            <p>
              {{ $t("item.serverFrameTime") }}:
              {{ serverMetrics?.server_frame_time }}(ms)
            </p>
            <p>
              {{ $t("item.maxPlayerNum") }}: {{ serverMetrics?.max_player_num }}
            </p>
          </div>
        </n-tooltip>
      </n-space>

      <n-space>
        <n-button
          type="default"
          secondary
          strong
          circle
          :title="isDarkMode ? '切换到亮色模式' : '切换到深色模式'"
          :aria-label="isDarkMode ? '切换到亮色模式' : '切换到深色模式'"
          @click="toggleDarkMode"
        >
          <template #icon>
            <n-icon>
              <svg v-if="isDarkMode" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="5"/>
                <line x1="12" y1="1" x2="12" y2="3"/>
                <line x1="12" y1="21" x2="12" y2="23"/>
                <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/>
                <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
                <line x1="1" y1="12" x2="3" y2="12"/>
                <line x1="21" y1="12" x2="23" y2="12"/>
                <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/>
                <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
              </svg>
              <svg v-else stroke="currentColor" fill="none" stroke-width="2" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round" xmlns="http://www.w3.org/2000/svg">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
              </svg>
            </n-icon>
          </template>
        </n-button>

        <n-dropdown
          trigger="hover"
          :options="languageOptions"
          @select="handleSelectLanguage"
        >
          <n-button
            type="default"
            secondary
            strong
            circle
            :aria-label="$t('button.language')"
          >
            <template #icon>
              <n-icon><LanguageSharp /></n-icon>
            </template>
          </n-button>
        </n-dropdown>

        <n-button
          type="primary"
          secondary
          strong
          @click="showLoginModal = true"
          v-if="!isLogin"
        >
          <template #icon>
            <n-icon>
              <AdminPanelSettingsOutlined />
            </n-icon>
          </template>
          {{ $t("button.auth") }}
        </n-button>
        <n-tag v-else type="success" size="large" round>
          <template #icon>
            <n-icon>
              <AdminPanelSettingsOutlined />
            </n-icon>
          </template>
          {{ $t("status.authenticated") }}
        </n-tag>
      </n-space>
    </div>
    <div class="w-full">
      <div class="rounded-lg" v-if="!loading">
        <n-layout style="height: calc(100vh - 64px)">
          <n-layout-header class="p-3 flex justify-between h-16" bordered>
            <n-button-group :size="smallScreen ? 'medium' : 'large'">
              <n-button
                v-if="isLogin"
                @click="toOverview"
                :type="currentDisplay === 'overview' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon><DashboardOutlined /></n-icon>
                </template>
                {{ $t("button.overview") }}
              </n-button>
              <n-button
                @click="toPlayers"
                :type="currentDisplay === 'players' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon>
                    <GameController />
                  </n-icon>
                </template>
                {{ $t("button.players") }}
              </n-button>
              <n-button
                @click="toGuilds()"
                :type="currentDisplay === 'guilds' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon>
                    <SupervisedUserCircleRound />
                  </n-icon>
                </template>
                {{ $t("button.guilds") }}
              </n-button>
              <n-button
                @click="toMap()"
                :type="currentDisplay === 'map' ? 'primary' : 'tertiary'"
                secondary
                strong
                round
              >
                <template #icon>
                  <n-icon>
                    <PublicRound />
                  </n-icon>
                </template>
                {{ $t("button.map") }}
              </n-button>
            </n-button-group>
            <n-space style="align-items:center">
              <n-tag type="info" round size="large" style="cursor:pointer" title="点击立即刷新"
                @click="getPlayerList()">{{
                $t("status.player_number", { number: playerList?.length })
              }}</n-tag>
              <n-tag type="success" round size="large" style="cursor:pointer" title="点击立即刷新"
                @click="getPlayerList()">{{
                $t("status.online_number", {
                  number:
                    serverMetrics?.current_player_num ??
                    onlinePlayerList?.length ??
                    0,
                })
              }}</n-tag>
              <n-button circle size="small" secondary title="立即刷新玩家数据"
                @click="getPlayerList(); getServerMetrics()">
                <template #icon>
                  <n-icon><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
                    stroke-linecap="round" stroke-linejoin="round" width="14" height="14">
                    <polyline points="23 4 23 10 17 10"/>
                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
                  </svg></n-icon>
                </template>
              </n-button>
            </n-space>
            <n-space v-if="isLogin" class="flex items-center">
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="success"
                secondary
                strong
                round
                @click="handleBackupList"
              >
                <template #icon>
                  <n-icon>
                    <ArchiveOutlined />
                  </n-icon>
                </template>
                {{ $t("button.backup") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="primary"
                secondary
                strong
                round
                @click="handleRconDrawer"
              >
                <template #icon>
                  <n-icon>
                    <Terminal />
                  </n-icon>
                </template>
                {{ $t("button.rcon") }}
              </n-button>
              <n-dropdown
                trigger="click"
                size="large"
                :options="controlCenterOption"
                @select="handleSelectControlCenter"
              >
                <n-button
                  :size="smallScreen ? 'medium' : 'large'"
                  type="error"
                  secondary
                  strong
                  round
                >
                  <template #icon>
                    <n-icon>
                      <GuiManagement />
                    </n-icon>
                  </template>
                  {{ $t("button.controlCenter") }}</n-button
                >
              </n-dropdown>
              <!-- <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="default"
                secondary
                strong
                round
                @click="toPalConf"
              >
                <template #icon>
                  <n-icon>
                    <Settings />
                  </n-icon>
                </template>
                {{ $t("button.palconf") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="warning"
                secondary
                strong
                round
                @click="handleWhiteList"
              >
                <template #icon>
                  <n-icon>
                    <ShieldCheckmarkSharp />
                  </n-icon>
                </template>
                {{ $t("button.whitelist") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="success"
                secondary
                strong
                round
                @click="handleStartBrodcast"
              >
                <template #icon>
                  <n-icon>
                    <BroadcastTower />
                  </n-icon>
                </template>
                {{ $t("button.broadcast") }}
              </n-button>
              <n-button
                :size="smallScreen ? 'medium' : 'large'"
                type="error"
                secondary
                strong
                round
                @click="handleShutdown"
              >
                <template #icon>
                  <n-icon>
                    <SettingsPowerRound />
                  </n-icon>
                </template>
                {{ $t("button.shutdown") }}
              </n-button> -->
            </n-space>
          </n-layout-header>
          <div class="overflow-hidden" style="height: calc(100% - 64px)">
            <admin-overview
              v-if="currentDisplay === 'overview'"
              :server-info="serverInfo"
              :server-metrics="serverMetrics"
              :players="playerList"
              @open-rcon="handleRconDrawer"
              @open-backup="handleBackupList"
              @open-broadcast="handleStartBrodcast"
              @open-config="handleSelectControlCenter('settings')"
              @open-game-config="handleGameConfig"
              @refresh-server="async () => { await getServerInfo(); await getServerMetrics(); }"
            />
            <player-list
              v-if="currentDisplay === 'players'"
              :players="playerList"
              @onWhitelistStatus="getSonWhitelistStatus"
            ></player-list>
            <item-manager
              v-if="currentDisplay === 'items'"
              :players="playerList"
            ></item-manager>
            <guild-list
              v-if="currentDisplay === 'guilds'"
              :guilds="guildList"
            ></guild-list>
            <map-view v-if="currentDisplay === 'map'"></map-view>
          </div>
        </n-layout>
      </div>
    </div>
  </div>
  <!-- login modal -->
  <n-modal
    v-model:show="showLoginModal"
    class="custom-card"
    preset="card"
    style="width: 90%; max-width: 600px"
    footer-style="padding: 12px;"
    content-style="padding: 12px;"
    header-style="padding: 12px;"
    :title="$t('modal.auth')"
    size="huge"
    :bordered="false"
  >
    <div>
      <span class="block pb-2">{{ $t("message.authdesc") }}</span>
      <n-input
        type="password"
        show-password-on="click"
        size="large"
        v-model:value="password"
        :aria-label="$t('modal.auth')"
        autocomplete="current-password"
        @keyup.enter="handleLogin"
      ></n-input>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <n-button
          type="tertiary"
          @click="
            () => {
              showLoginModal = false;
              password = '';
            }
          "
          >{{ $t("button.cancel") }}</n-button
        >
        <n-button class="ml-3 w-40" type="primary" @click="handleLogin">{{
          $t("button.confirm")
        }}</n-button>
      </div>
    </template>
  </n-modal>

  <rcon-manager v-model:show="showRconDrawer" />
  <broadcast-composer v-model:show="showBroadcastModal" />
  <shutdown-dialog v-model:show="showShutdownDialog" />
  <backup-manager v-model:show="backupModal" />
  <game-config-modal v-model:show="showGameConfigModal" />
  <whitelist-manager
    v-model:show="showWhiteListModal"
    :players="playerList"
    @updated="getSonWhitelistStatus"
  />
  <setup-wizard />
</template>
