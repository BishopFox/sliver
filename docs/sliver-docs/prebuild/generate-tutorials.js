const fs = require('fs/promises');
const path = require('path');

const workingDirectory = process.cwd();
const directoryPath = path.join(workingDirectory, 'pages/tutorials/md');

async function generateTutorials() {
    let entries;
    try {
        entries = await fs.readdir(directoryPath, { withFileTypes: true });
    } catch (error) {
        if (error.code === 'ENOENT') {
            await fs.writeFile(
                path.join(workingDirectory, 'public/tutorials.json'),
                JSON.stringify({ tutorials: [] })
            );
            return;
        }
        throw error;
    }
    const files = entries
        .filter((entry) => entry.isFile() && entry.name.endsWith('.md'))
        .map((entry) => entry.name);

    const tutorials = [];
    for (const file of files) {
        const filePath = path.join(directoryPath, file);
        const fileContent = await fs.readFile(filePath, 'utf8');
        const name = path.basename(file, '.md');
        tutorials.push({
            name,
            content: fileContent,
        });
    }

    await fs.writeFile(
        path.join(workingDirectory, 'public/tutorials.json'),
        JSON.stringify({ tutorials })
    );
}

module.exports = generateTutorials;

if (require.main === module) {
    generateTutorials().catch((error) => {
        console.error('[prebuild] Failed to generate tutorials.json', error);
        process.exitCode = 1;
    });
}
