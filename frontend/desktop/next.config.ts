import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // The dev badge sat on top of the sidebar footer in the bottom-left corner.
  devIndicators: { position: "bottom-right" },
  // Electron serves this export from app://prova/ in production.
  output: "export",
  trailingSlash: true,
  images: { unoptimized: true },
  turbopack: {
    root: path.join(__dirname, ".."),
  },
  experimental: {
    externalDir: true,
  },
};

export default nextConfig;
