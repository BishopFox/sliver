const fs = require('fs/promises');
const path = require('path');

const workingDirectory = process.cwd();
const directoryPath = `${workingDirectory}/pages/tutorials/md`;

async function generateSiteMap() {
    const ls = await fs.readdir(directoryPath);
    const files = ls.filter((file) => file.endsWith('.md'));
    const tutorials = [];
    for (const file of files) {
        const filePath = path.join(directoryPath, file);
        const fileContent = await fs.readFile(filePath, 'utf8');
        const name = path.basename(file).replace('.md', '');
        tutorials.push({
            name: name,
            content: fileContent,
        });
    };
    return {tutorials: tutorials};
}

generateSiteMap().then(async (sitemap) => {
    await fs.writeFile(`${workingDirectory}/public/tutorials.json`, JSON.stringify(sitemap));
});
