import nextCoreWebVitals from "eslint-config-next/core-web-vitals";

const config = [
  ...nextCoreWebVitals,
  {
    // Monaco bundles are vendor artifacts, not authored source files.
    ignores: ["public/js/**"],
  },
];

export default config;
