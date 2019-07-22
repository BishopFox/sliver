import * as fs from 'fs';
import * as path from 'path';


type ProtocolCallback = (arg0: { mimeType: string; data: Buffer; }) => void;
const DIST_PATH = path.join(__dirname, 'dist');
export const scheme = 'app';


const mimeTypes = {
  '.js': 'text/javascript',
  '.mjs': 'text/javascript',
  '.html': 'text/html',
  '.htm': 'text/html',
  '.json': 'application/json',
  '.css': 'text/css',
  '.svg': 'application/svg+xml',
  '.ico': 'image/vnd.microsoft.icon',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.map': 'text/plain'
};


function mime(filename: string): string {
  const type = mimeTypes[path.extname(`${filename || ''}`).toLowerCase()];
  return type ? type : 'application/octet-stream';
}

export function requestHandler(req: Electron.RegisterBufferProtocolRequest, next: ProtocolCallback) {
  const reqUrl = new URL(req.url);
  console.log('[app-protocol] URL:');
  console.log(reqUrl);
  let reqPath = path.normalize(reqUrl.pathname);
  if (reqPath === '/') {
    reqPath = '/index.html';
  }
  const reqFilename = path.basename(reqPath);
  console.log(`[app-protocol] read: ${path.join(DIST_PATH, reqPath)}`);
  fs.readFile(path.join(DIST_PATH, reqPath), (err, data) => {
    if (!err) {
      next({
        mimeType: mime(reqFilename),
        data: data
      });
    } else {
      console.error(err);
    }
  });
}
