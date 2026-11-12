import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";
import { previewApi } from "../../tooling/preview.ts";
import { heroImageSizes } from "./src/hero-image-settings";
export default defineConfig({
  root: import.meta.dirname,
  envDir: fileURLToPath(new URL("../..", import.meta.url)),
  plugins: [
    react(),
    previewApi(),
    {
      name: "visitor-hero-preload",
      apply: "build",
      transformIndexHtml: {
        order: "post",
        handler(html, context) {
          const sources = Object.values(context.bundle ?? {})
            .filter(
              (asset) => asset.type === "asset" && /\/nusantara-\d+-.*\.avif$/.test(asset.fileName),
            )
            .map((asset) => ({
              url: `/${asset.fileName}`,
              width: Number(asset.fileName.match(/nusantara-(\d+)-/)![1]),
            }))
            .sort((a, b) => a.width - b.width);
          if (sources.length !== 4) throw new Error("Visitor hero preload assets are missing.");
          const srcset = sources.map(({ url, width }) => `${url} ${width}w`).join(", ");
          // Run after viewport metadata, before stylesheets can block the preload script.
          return html.replace(
            "<!-- visitor-hero-preload -->",
            `<script>if(location.pathname==="/"){const l=document.createElement("link");l.rel="preload";l.as="image";l.type="image/avif";l.imageSrcset=${JSON.stringify(srcset)};l.imageSizes=${JSON.stringify(heroImageSizes)};l.fetchPriority="high";document.head.append(l)}</script>`,
          );
        },
      },
    },
  ],
  server: {
    fs: {
      allow: [fileURLToPath(new URL("../..", import.meta.url))],
    },
    proxy: {
      "/admin": { target: "http://127.0.0.1:3002", ws: true },
      "/v1": { target: "http://127.0.0.1:8080" },
    },
  },
  build: { outDir: "dist" },
});
