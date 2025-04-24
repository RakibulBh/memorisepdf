import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  api: {
    bodyParser: {
      sizeLimit: "100mb",
    },
    responseLimit: "100mb",
  },
};

// file size limit to 100mb
export default nextConfig;
