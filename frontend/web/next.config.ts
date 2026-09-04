import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // The dev badge sat on top of the sidebar footer in the bottom-left corner.
  devIndicators: { position: "bottom-right" },
  // Shared components live under frontend/shared; the root is one level up.
  turbopack: {
    root: path.join(__dirname, ".."),
  },
  experimental: {
    externalDir: true,
  },
};

export default nextConfig;
