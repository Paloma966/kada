import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  // NEXT_PUBLIC_API_URL is deliberately not hardcoded here.
  //
  // In production the frontend is served by nginx on the same origin as the API, so the variable is
  // empty there and requests stay relative. Declaring that empty value in `env` applied it to local
  // development as well, where it beat the fallback in src/lib/api.ts (an empty string is not nullish)
  // and sent every request to the Next.js dev server, which answered with an HTML 404.
  //
  // Local development now uses the fallback in src/lib/api.ts (http://localhost:8080); a deployment
  // that needs a different base URL passes NEXT_PUBLIC_API_URL at build time, for example in
  // frontend/.env.production.
};

export default nextConfig;
