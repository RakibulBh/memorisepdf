import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Produces a server bundle carrying only the dependencies actually reached,
  // which is what the Dockerfile's run stage copies.
  output: "standalone",

  // Lecture decks run to a hundred megabytes, far past the default body limit.
  experimental: {
    serverActions: {
      bodySizeLimit: "100mb",
    },
  },

  async rewrites() {
    // In development the API runs on its own port. In production the proxy
    // routes these paths, so no rewrite is needed and the base URL is empty.
    const api = process.env.API_ORIGIN;
    if (!api) return [];

    return [
      { source: "/api/:path*", destination: `${api}/api/:path*` },
      { source: "/auth/:path*", destination: `${api}/auth/:path*` },
    ];
  },
};

export default nextConfig;
