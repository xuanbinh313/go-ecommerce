import inject from "@medusajs/admin-vite-plugin"
import { VIRTUAL_MODULES } from "@medusajs/admin-shared"
import react from "@vitejs/plugin-react"
import { defineConfig, loadEnv } from "vite"
import { fileURLToPath } from "node:url"

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd())
  return {
    plugins: [react(), inject({ sources: [] })],
    optimizeDeps: {
      include: [
        "react",
        "react/jsx-runtime",
        "react-dom/client",
        "react-router-dom",
        "react-router",
        "react-router/dom",
        "react-i18next",
        "@medusajs/ui",
        "@medusajs/dashboard",
        "@medusajs/js-sdk",
        "@tanstack/react-query",
        "copy-to-clipboard",
        "lodash.isequal",
        "void-elements",
        "qs",
        "cookie",
        "set-cookie-parser",
      ],
      exclude: [...VIRTUAL_MODULES],
    },
    resolve: {
      alias: [
        {
          find: /^use-sync-external-store\/shim(?:\/index\.js)?$/,
          replacement: fileURLToPath(new URL("./src/shims/use-sync-external-store.ts", import.meta.url)),
        },
      ],
    },
    define: {
      __BASE__: JSON.stringify(env.VITE_MEDUSA_BASE || "/"),
      __BACKEND_URL__: JSON.stringify(env.VITE_MEDUSA_BACKEND_URL || "http://localhost:8080/api/v1"),
      __STOREFRONT_URL__: JSON.stringify(env.VITE_MEDUSA_STOREFRONT_URL || "http://localhost:8000"),
      __AUTH_TYPE__: JSON.stringify("jwt"),
      __JWT_TOKEN_STORAGE_KEY__: JSON.stringify("micro_ecommerce_admin_jwt"),
    },
  }
})
