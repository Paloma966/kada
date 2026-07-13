import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  env: {
    NEXT_PUBLIC_API_URL: "", // 同源，Nginx 代理 /api/ → Go 后端
  },
};

export default nextConfig;
