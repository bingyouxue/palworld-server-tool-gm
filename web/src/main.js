import "./assets/common.less";

import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import router from "./router";

import "virtual:uno.css";

import i18n from "@/assets/i18n.js";

const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(i18n);

app.directive("click-outside", {
  beforeMount(el, binding) {
    el._clickOutsideHandler = (e) => {
      if (!el.contains(e.target)) binding.value(e);
    };
    document.addEventListener("mousedown", el._clickOutsideHandler);
  },
  unmounted(el) {
    document.removeEventListener("mousedown", el._clickOutsideHandler);
  },
});

app.mount("#app");
