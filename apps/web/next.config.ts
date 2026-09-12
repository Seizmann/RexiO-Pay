import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // shared-types ships TypeScript source; compile it as part of the app.
  transpilePackages: ["@rexio-pay/shared-types"],
};

export default nextConfig;
