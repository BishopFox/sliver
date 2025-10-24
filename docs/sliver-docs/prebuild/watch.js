const fs = require('fs');
const path = require('path');
const runPrebuild = require('./run-prebuild');

const DOCS_DIR = path.join(process.cwd(), 'pages/docs/md');
const TUTORIALS_DIR = path.join(process.cwd(), 'pages/tutorials/md');
const args = process.argv.slice(2);
const skipInitial = args.includes('--skip-initial');

let isRunning = false;
let isQueued = false;

async function triggerPrebuild(reason) {
  if (isRunning) {
    isQueued = true;
    return;
  }

  isRunning = true;
  const label = `[prebuild] ${reason}`;
  console.log(`${label} -> generating docs.json & tutorials.json`);
  const start = Date.now();
  try {
    await runPrebuild();
    const duration = Date.now() - start;
    console.log(`[prebuild] completed in ${duration}ms`);
  } catch (error) {
    console.error('[prebuild] failed', error);
  } finally {
    isRunning = false;
    if (isQueued) {
      isQueued = false;
      triggerPrebuild('queued change');
    }
  }
}

function watchMarkdown(dir) {
  if (!fs.existsSync(dir)) {
    return null;
  }

  const watcher = fs.watch(dir, { persistent: true }, (eventType, filename) => {
    if (!filename || !filename.endsWith('.md')) {
      return;
    }
    const reason = `${path.relative(process.cwd(), path.join(dir, filename))} ${eventType}`;
    triggerPrebuild(reason);
  });

  watcher.on('error', (error) => {
    console.error(`[prebuild] watcher error for ${dir}`, error);
  });

  console.log(`[prebuild] watching ${dir}`);
  return watcher;
}

async function watchPrebuild() {
  if (skipInitial) {
    console.log('[prebuild] startup run skipped (skip-initial flag)');
  } else {
    await triggerPrebuild('startup');
  }
  const watchers = [watchMarkdown(DOCS_DIR), watchMarkdown(TUTORIALS_DIR)].filter(Boolean);

  const shutdown = () => {
    watchers.forEach((watcher) => watcher.close());
  };

  process.on('SIGINT', () => {
    shutdown();
    process.exit(0);
  });

  process.on('SIGTERM', () => {
    shutdown();
    process.exit(0);
  });

  return () => shutdown();
}

module.exports = watchPrebuild;

if (require.main === module) {
  watchPrebuild().catch((error) => {
    console.error('[prebuild] watcher bootstrap failed', error);
    process.exit(1);
  });
}
