import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { resolve } from "node:path";

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: "../public",
    emptyOutDir: false,
    rollupOptions: {
      input: {
        portal: resolve(__dirname, "portal.html"),
        managers: resolve(__dirname, "managers.html"),
        models: resolve(__dirname, "models.html"),
      },
    },
  },
});
