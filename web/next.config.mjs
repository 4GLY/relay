/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Emit a self-contained `.next/standalone` bundle so the Docker runtime
  // can ship a single `node server.js` entrypoint.
  output: "standalone",
};

export default nextConfig;
