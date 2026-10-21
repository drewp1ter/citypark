import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Dev only: hosts other than localhost that may open the dev server. Without this, Next 16
  // blocks the HMR connection and the page never hydrates (buttons like ThemeToggle do nothing).
  allowedDevOrigins: ["gpubox", "172.19.0.2"],
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
