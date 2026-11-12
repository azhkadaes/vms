import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";
export default defineConfig({
  root: import.meta.dirname,
  envDir: fileURLToPath(new URL("../..", import.meta.url)),
  base: "/admin/",
  plugins: [react()],
  optimizeDeps: { include: ["react-hook-form", "@hookform/resolvers/zod"] },
  server: {
    fs: {
      allow: [fileURLToPath(new URL("../..", import.meta.url))],
    },
    proxy: {
      "/__preview": { target: "http://127.0.0.1:3000" },
      "/v1": { target: "http://127.0.0.1:8080" },
    },
  },
  build: { outDir: "dist" },
});
