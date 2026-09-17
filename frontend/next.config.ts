import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // API 请求在本地开发也走同源：Next dev server 把 /api/* 代理到 Go 后端，
  // 和生产的 nginx 同源代理行为一致，前端代码里没有跨域、没有 localhost:8080 硬编码。
  // 生产环境 nginx 会在更外层直接代理 /api/* 到 backend，这里的 rewrites 不会触发。
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: "http://localhost:8080/api/:path*",
      },
    ];
  },
};

export default nextConfig;
