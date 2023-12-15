/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export',
  reactStrictMode: true,
  images: {
    unoptimized: true,
  },
  webpack: (config, { isServer }) => {
    if (isServer) {
      require('./prebuild/generate-docs');
    }
    return config;
  }
}

module.exports = nextConfig;
