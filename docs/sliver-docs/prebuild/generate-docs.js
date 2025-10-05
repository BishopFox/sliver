const fs = require('fs/promises');
const path = require('path');

const workingDirectory = process.cwd();
const directoryPath = path.join(workingDirectory, 'pages/docs/md');

async function generateDocs() {
    let entries;
    try {
        entries = await fs.readdir(directoryPath, { withFileTypes: true });
    } catch (error) {
        if (error.code === 'ENOENT') {
            await fs.writeFile(
                path.join(workingDirectory, 'public/docs.json'),
                JSON.stringify({ docs: [] })
            );
            return;
        }
        throw error;
    }
    const files = entries
        .filter((entry) => entry.isFile() && entry.name.endsWith('.md'))
        .map((entry) => entry.name);

    const docs = [];
    for (const file of files) {
        const filePath = path.join(directoryPath, file);
        const fileContent = await fs.readFile(filePath, 'utf8');
        const name = path.basename(file, '.md');
        docs.push({
            name,
            content: fileContent,
        });
    }

    await fs.writeFile(
        path.join(workingDirectory, 'public/docs.json'),
        JSON.stringify({ docs })
    );
}

module.exports = generateDocs;

if (require.main === module) {
    generateDocs().catch((error) => {
        console.error('[prebuild] Failed to generate docs.json', error);
        process.exitCode = 1;
    });
}
