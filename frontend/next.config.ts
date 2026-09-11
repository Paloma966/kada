import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  env: {
    NEXT_PUBLIC_API_URL: "", // Same origin — Nginx proxies /api/ to the Go backend
  },
};

export default nextConfig;
