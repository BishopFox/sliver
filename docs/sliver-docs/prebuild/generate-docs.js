const fs = require('fs/promises');
const path = require('path');

const workingDirectory = process.cwd();
const directoryPath = `${workingDirectory}/pages/docs/md`;

async function generateSiteMap() {
    const ls = await fs.readdir(directoryPath);
    const files = ls.filter((file) => file.endsWith('.md'));
    const docs = [];
    for (const file of files) {
        const filePath = path.join(directoryPath, file);
        const fileContent = await fs.readFile(filePath, 'utf8');
        const name = path.basename(file).replace('.md', '');
        docs.push({
            name: name,
            content: fileContent,
        });
    };
    return {docs: docs};
}

generateSiteMap().then(async (sitemap) => {
    await fs.writeFile(`${workingDirectory}/public/docs.json`, JSON.stringify(sitemap));
});
