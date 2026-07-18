import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import vueJsx from "@vitejs/plugin-vue-jsx";
import UnoCSS from "unocss/vite";
import fs from "fs";
import path from "path";

import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { NaiveUiResolver } from "unplugin-vue-components/resolvers";
import { VantResolver } from "unplugin-vue-components/resolvers";
import { fileURLToPath } from "url";

// Removes stale hashed JS/CSS bundles from the assets dir before each build
// so the Go embed only ever picks up one copy of each chunk.
function cleanHashedAssets() {
  return {
    name: "clean-hashed-assets",
    buildStart() {
      const assetsDir = path.resolve(__dirname, "../assets");
      if (!fs.existsSync(assetsDir)) return;
      const stalePattern = /^(Home|index|leaflet-src[^/]*)-[A-Za-z0-9_\-]+\.(js|css)$/;
      let removed = 0;
      for (const f of fs.readdirSync(assetsDir)) {
        if (stalePattern.test(f)) {
          fs.rmSync(path.join(assetsDir, f));
          removed++;
        }
      }
      if (removed) console.log(`[clean-hashed-assets] removed ${removed} stale files`);
    },
  };
}

// const debugMode = process.env.APP_ENV !== 'prod'

export default defineConfig({
  base: "/",
  build: {
    outDir: "../",
    emptyOutDir: false,
    rollupOptions: {
      output: {
        // All hashed JS/CSS go into assets/ — cleaned by the pre-build hook below
        assetFileNames: "assets/[name]-[hash][extname]",
        chunkFileNames: "assets/[name]-[hash].js",
        entryFileNames: "assets/[name]-[hash].js",
      },
    },
  },
  plugins: [
    cleanHashedAssets(),
    vue(),
    vueJsx(),
    AutoImport({
      imports: [
        "vue",
        {
          "naive-ui": [
            "useDialog",
            "useMessage",
            "useNotification",
            "useLoadingBar",
          ],
        },
      ],
    }),
    Components({
      resolvers: [NaiveUiResolver(), VantResolver()],
    }),
    UnoCSS(),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: process.env.VITE_API_PROXY_TARGET || "http://127.0.0.1:8080",
        changeOrigin: true,
        // rewrite: (path) => path.replace(/^\/api/, ""),
      },
      "/map": {
        target: process.env.VITE_API_PROXY_TARGET || "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
});
