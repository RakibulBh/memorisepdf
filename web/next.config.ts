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
    // Development only. Next resolves rewrites when it builds, so this cannot
    // be changed at runtime -- and it does not need to be: in production the
    // reverse proxy routes /api and /auth to the api container, and the
    // production image is built with API_ORIGIN unset so no rewrite exists.
    const api = process.env.API_ORIGIN;
    if (!api) return [];

    return [
      { source: "/api/:path*", destination: `${api}/api/:path*` },
      { source: "/auth/:path*", destination: `${api}/auth/:path*` },
    ];
  },
};

export default nextConfig;
