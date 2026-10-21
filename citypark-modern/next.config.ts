import type { NextConfig } from "next";

// Host of the menu photos (from iiko); without it remote photos are not used at all.
const menuImageHost = process.env.MENU_IMAGE_HOST;

const nextConfig: NextConfig = {
  // Dev only: hosts other than localhost that may open the dev server. Without this, Next 16
  // blocks the HMR connection and the page never hydrates (buttons like ThemeToggle do nothing).
  allowedDevOrigins: ["gpubox", "172.19.0.2"],
  images: menuImageHost ? { remotePatterns: [{ protocol: "https", hostname: menuImageHost }] } : undefined,
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
