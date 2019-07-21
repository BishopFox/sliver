import * as fs from 'fs';
import * as path from 'path';


type ProtocolCallback = (arg0: { mimeType: string; data: Buffer; }) => void;
const DIST_PATH = path.join(__dirname, 'dist');
export const scheme = 'app';


const mimeTypes = {
  '.js': 'text/javascript',
  '.ts': 'text/javascript',
  '.mjs': 'text/javascript',
  '.html': 'text/html',
  '.htm': 'text/html',
  '.json': 'application/json',
  '.css': 'text/css',
  '.svg': 'application/svg+xml',
};


function mime(filename: string): string {
  return mimeTypes[path.extname(`${filename || ''}`).toLowerCase()];
}

export function requestHandler(req: Electron.RegisterBufferProtocolRequest, next: ProtocolCallback) {
  const reqUrl = new URL(req.url);
  console.log('[app-protocol] URL:');
  console.log(reqUrl);
  const reqPath = path.normalize(reqUrl.pathname);
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
