const fs = require('fs/promises');
const path = require('path');

const workingDirectory = process.cwd();
const versionDir = path.join(workingDirectory, 'util/__generated__');
const versionFile = path.join(versionDir, 'prebuild-version.ts');

async function updateVersion() {
  await fs.mkdir(versionDir, { recursive: true });
  const version = Date.now().toString();
  const contents = `export const PREBUILD_VERSION = ${JSON.stringify(version)} as const;\n`;
  await fs.writeFile(versionFile, contents, 'utf8');
}

module.exports = updateVersion;

if (require.main === module) {
  updateVersion().catch((error) => {
    console.error('[prebuild] Failed to write prebuild version', error);
    process.exitCode = 1;
  });
}
