<script setup>
import { zhCN, dateZhCN, jaJP, dateJaJP, darkTheme } from "naive-ui";
import pageStore from "@/stores/model/page.js";
import { onMounted, provide } from "vue";

// Persist dark mode preference to localStorage; respect system default on first load
const storedDark = localStorage.getItem("pst_dark_mode");
const isDarkMode = ref(
  storedDark !== null
    ? storedDark === "1"
    : window.matchMedia("(prefers-color-scheme: dark)").matches,
);

const toggleDarkMode = () => {
  isDarkMode.value = !isDarkMode.value;
  localStorage.setItem("pst_dark_mode", isDarkMode.value ? "1" : "0");
};

// Sync if system preference changes and user hasn't set a manual preference
const updateDarkMode = (e) => {
  if (localStorage.getItem("pst_dark_mode") === null) {
    isDarkMode.value = e.matches;
  }
};

// Expose to all descendants
provide("isDarkMode", isDarkMode);
provide("toggleDarkMode", toggleDarkMode);

const themeOverrides = {
  common: {
    primaryColor: "#4098fc",
    primaryColorHover: "#4098fc",
  },
};

const locale = ref(null);
const uiLocale = ref(null);
const uiDateLocale = ref(null);

let getScreenWidth = function () {
  let scrollWidth = document.documentElement.clientWidth || window.innerWidth;
  pageStore().setScreenWidth(scrollWidth);
};

onMounted(() => {
  const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
  mediaQuery.addEventListener("change", updateDarkMode);
  getScreenWidth();
  window.onresize = function () {
    getScreenWidth();
  };

  let localLocale = localStorage.getItem("locale");
  if (localLocale) {
    locale.value = localLocale;
    if (locale.value == "zh") {
      uiLocale.value = zhCN;
      uiDateLocale.value = dateZhCN;
    } else if (locale.value == "ja") {
      uiLocale.value = jaJP;
      uiDateLocale.value = dateJaJP;
    } else if (locale.value == "en") {
      uiLocale.value = null;
      uiDateLocale.value = null;
    }
  } else {
    localStorage.setItem("locale", "zh");
    locale.value = "zh";
    uiLocale.value = zhCN;
    uiDateLocale.value = dateZhCN;
  }
});
</script>

<template>
  <n-config-provider
    :locale="uiLocale"
    :date-locale="uiDateLocale"
    :theme-overrides="themeOverrides"
    :theme="isDarkMode ? darkTheme : null"
  >
    <n-dialog-provider>
      <n-notification-provider>
        <n-message-provider>
          <router-view />
        </n-message-provider>
      </n-notification-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>
