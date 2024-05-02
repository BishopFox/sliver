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
      require('./prebuild/generate-docs');
      require('./prebuild/generate-tutorials');
    }
    return config;
  }
}

module.exports = nextConfig;
