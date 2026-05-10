import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "path";

export default defineConfig({
  base: "/ui/",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "../internal/uifs/dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api/v1": {
        target: "http://localhost:7777",
        changeOrigin: true,
      },
    },
  },
});
