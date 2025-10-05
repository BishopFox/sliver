const generateDocs = require('./generate-docs');
const generateTutorials = require('./generate-tutorials');
const updateVersion = require('./update-version');

async function runPrebuild() {
  await generateDocs();
  await generateTutorials();
  await updateVersion();
}

module.exports = runPrebuild;

if (require.main === module) {
  runPrebuild().catch((error) => {
    console.error('[prebuild] Failed to run prebuild scripts', error);
    process.exitCode = 1;
  });
}
