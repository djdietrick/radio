import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// During dev, proxy /api and /healthz to the Go backend so the frontend can use
// same-origin relative URLs. In production the built static files are served by
// nginx and point at the backend via the API base.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": { target: "http://localhost:8080", changeOrigin: true },
      "/healthz": { target: "http://localhost:8080", changeOrigin: true },
    },
  },
});
