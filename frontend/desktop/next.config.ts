import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // The dev badge sat on top of the sidebar footer in the bottom-left corner.
  devIndicators: { position: "bottom-right" },
  // Electron loads the static export over file://.
  output: "export",
  trailingSlash: true,
  // Asset paths must be relative; absolute paths break under file://.
  // Production output only: a relative prefix breaks CSS on the dev server.
  assetPrefix: process.env.NODE_ENV === "production" ? "./" : undefined,
  images: { unoptimized: true },
  turbopack: {
    root: path.join(__dirname, ".."),
  },
  experimental: {
    externalDir: true,
  },
};

export default nextConfig;
