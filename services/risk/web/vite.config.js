import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const proxyTarget = process.env.VITE_PROXY_TARGET || "http://sub2api-risk-api:8091";
const basePath = process.env.VITE_BASE_PATH || "/";

export default defineConfig({
  base: basePath,
  plugins: [react()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    allowedHosts: ["ai.gunddam.dpdns.org"],
    proxy: {
      "/api": {
        target: proxyTarget,
        changeOrigin: true,
      },
      "/healthz": {
        target: proxyTarget,
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: "0.0.0.0",
    port: 4173,
    allowedHosts: ["ai.gunddam.dpdns.org"],
  },
});
