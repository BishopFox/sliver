const { spawnSync } = require('child_process');
const path = require('path');

/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  reactStrictMode: true,
  images: {
    unoptimized: true,
  },
  experimental: {
    scrollRestoration: true,
  },
  webpack: (config, { isServer }) => {
    if (isServer) {
      const result = spawnSync(process.execPath, [path.join(__dirname, 'prebuild/run-prebuild.js')], {
        stdio: 'inherit',
      });

      if (result.status !== 0) {
        throw new Error('[prebuild] webpack hook failed');
      }
    }
    return config;
  }
}

module.exports = nextConfig;
