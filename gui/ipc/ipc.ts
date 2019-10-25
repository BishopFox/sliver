/*
  Sliver Implant Framework
  Copyright (C) 2019  Bishop Fox
  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.
  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.
  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
--------------------------------------------------------------------------

Maps IPC calls to RPC calls, and provides other local operations such as
listing/selecting configs to the sandboxed code.

*/

import { ipcMain, dialog, FileFilter, BrowserWindow, IpcMainEvent } from 'electron';
import { homedir } from 'os';
import * as base64 from 'base64-arraybuffer';
import * as fs from 'fs';
import * as path from 'path';
import * as uuid from 'uuid';

import { jsonSchema } from './json-schema';
import { RPCClient, RPCConfig } from '../rpc';
import { Envelope } from '../rpc/pb';
import { rejects } from 'assert';


const CLIENT_DIR = path.join(homedir(), '.sliver-client');
const CONFIG_DIR = path.join(CLIENT_DIR, 'configs');
const SETTINGS_PATH = path.join(CLIENT_DIR, 'gui-settings.json');


let rpc: RPCClient;


function decodeRequest(data: string): Uint8Array {
  const buf = base64.decode(data);
  return new Uint8Array(buf);
}

export interface SaveFileReq {
  title: string;
  message: string;
  filename: string;
  data: string;
}

export interface ReadFileReq {
  title: string;
  message: string;
  openDirectory: boolean;
  multiSelections: boolean;
  filters: FileFilter[] | null; // { filters: [ { name: 'Custom File Type', extensions: ['as'] } ] }
}

export interface IPCMessage {
  id: number;
  type: string;
  method: string; // Identifies the target method and in the response if the method call was a success/error
  data: string;
}

export interface ScriptReq {
  devtools: boolean;
  script: string;
}

const sliverScriptParentURI = 'data:text/html;charset=utf-8,' + encodeURIComponent(`
<head>
  <meta http-equiv="Content-Security-Policy" content="default-src none; script-src app://sliver data:; frame-src data:">
</head>
<body></body>
`);

function renderDataURI(script: string) {
  return 'data:text/html;charset=utf-8,' + encodeURIComponent(`
<head>
  <meta http-equiv="Content-Security-Policy" content="default-src none; script-src app://sliver data:;">
  <script src="app://sliver/sliver-script/require.js"></script>
  <script src="app://sliver/sliver-script/rxjs.umd.min.js"></script>
  <script src="app://sliver/sliver-script/protobuf.min.js"></script>
  <script src="app://sliver/sliver-script/pb/constants.js"></script>
  <script src="app://sliver/sliver-script/api.js"></script>
  <script src="data:text/javascript;base64,${Buffer.from(script).toString('base64')}"></script>
</head>
`);
}


async function makeConfigDir(): Promise<NodeJS.ErrnoException|null> {
  return new Promise((resolve, reject) => {
    const dirOptions = {
      mode: 0o700, 
      recursive: true
    };
    fs.mkdir(CONFIG_DIR, dirOptions, (err) => {
      err ? reject(err) : resolve(null);
    });
  });
}


// IPC Methods used to start/interact with the RPCClient
export class IPCHandlers {

  @jsonSchema({
    "properties": {
      "operator": {"type": "string", "minLength": 1},
      "lhost": {"type": "string", "minLength": 1},
      "lport": {"type": "number"},
      "ca_certificate": {"type": "string", "minLength": 1},
      "certificate": {"type": "string", "minLength": 1},
      "private_key": {"type": "string", "minLength": 1},
    },
    "required": [
      "operator", "lhost", "lport", "ca_certificate", "certificate", "private_key"
    ]
  })
  static async client_start(req: string): Promise<string> {
    const config: RPCConfig = JSON.parse(req);
    rpc = new RPCClient(config);
    await rpc.connect();
    console.log('Connection successful');
    rpc.envelopeSubject$.subscribe((envelope) => {
      if (envelope.getId() === 0) {
        ipcMain.emit('push', base64.encode(envelope.serializeBinary()));
      }
    });
    return 'success';
  }

  static async client_executeScript(req: string): Promise<string> {
    const scriptReq: ScriptReq = JSON.parse(req);
    const scriptId: string = uuid.v4();
    const scriptWindow = new BrowserWindow({
      webPreferences: {
        sandbox: true,
        webSecurity: true,
        contextIsolation: true,
        webviewTag: false,
        enableRemoteModule: false,
        allowRunningInsecureContent: false,
        nodeIntegration: false,
        nodeIntegrationInWorker: false,
        nodeIntegrationInSubFrames: false,
        nativeWindowOpen: false,
        safeDialogs: true,

        preload: path.join(__dirname, '..', 'sliver-script', 'preload.js'),
        additionalArguments: [`--scriptId=${scriptId}`]
      },
      show: false,
    });

    scriptWindow.loadURL(sliverScriptParentURI).then(() => {
      scriptWindow.webContents.executeJavaScript(`var ScriptSrc = '${renderDataURI(scriptReq.script)}'`);
      scriptWindow.webContents.executeJavaScript(`
      const childFrame = document.createElement('iframe');
      childFrame.setAttribute('src', ScriptSrc);
      childFrame.setAttribute('sandbox', 'allow-scripts');
      document.body.appendChild(childFrame);
      `);
      if (scriptReq.devtools) {
        scriptWindow.webContents.openDevTools({ mode: 'detach' });
      }
    });
    return scriptId;
  }

  static async client_activeConfig(): Promise<string> {
    return rpc ? JSON.stringify(rpc.config) : '';
  }

  @jsonSchema({
    "properties": {
      "title": {"type": "string", "minLength": 1, "maxLength": 100},
      "message": {"type": "string", "minLength": 1, "maxLength": 100},
      "openDirectory": {"type": "boolean"},
      "multiSelections": {"type": "boolean"},
      "filter": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": {"type": "string"},
            "extensions": {
              "type": "array",
              "items": {"type": "string"}
            }
          }
        }
      }
    },
    "required": ["title", "message"]
  })
  static async client_readFile(req: string): Promise<string> {
    const readFileReq: ReadFileReq = JSON.parse(req);
    const dialogOptions = {
      title: readFileReq.title,
      message: readFileReq.message,
      openDirectory: readFileReq.openDirectory,
      multiSelections: readFileReq.multiSelections
    };
    const files = [];
    const open = await dialog.showOpenDialog(null, dialogOptions);
    await Promise.all(open.filePaths.map((filePath) => {
      return new Promise(async (resolve) => {
        fs.readFile(filePath, (err, data) => {
          files.push({
            filePath: filePath,
            error: err ? err.toString() : null,
            data: data ? base64.encode(data) : null
          });
          resolve();
        });
      });
    }));
    return JSON.stringify({ files: files });
  }

  @jsonSchema({
    "properties": {
      "title": {"type": "string", "minLength": 1, "maxLength": 100},
      "message": {"type": "string", "minLength": 1, "maxLength": 100},
      "filename": {"type": "string", "minLength": 1},
      "data": {"type": "string"}
    },
    "required": ["title", "message", "filename", "data"]
  })
  static client_saveFile(req: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      const saveFileReq: SaveFileReq = JSON.parse(req);
      const dialogOptions = {
        title: saveFileReq.title,
        message: saveFileReq.message,
        defaultPath: path.join(homedir(), 'Downloads', path.basename(saveFileReq.filename)),
      };
      const save = await dialog.showSaveDialog(dialogOptions);
      console.log(`[save file] ${save.filePath}`);
      if (save.canceled) {
        return resolve('');  // Must return to stop execution
      }
      const fileOptions = {
        mode: 0o644,
        encoding: 'binary',
      };
      const data = Buffer.from(base64.decode(saveFileReq.data));
      fs.writeFile(save.filePath, data, fileOptions, (err) => {
        if (err) {
          reject(err);
        } else {
          resolve(JSON.stringify({ filename: save.filePath }));
        }
      });
    });
  }

  static client_getSettings(): Promise<string> {
    return new Promise((resolve, reject) => {
      try {
        if (!fs.existsSync(SETTINGS_PATH)) {
          return resolve('{}');
        }
        fs.readFile(SETTINGS_PATH, 'utf-8', (err, data) => {
          if (err) {
            return reject(err);
          }
          JSON.parse(data);
          resolve(data);
        });
      } catch (err) {
        reject(err);
      }
    });
  }

  // The Node process never interacts with the "settings" values, so
  // we do not validate them, aside from ensuing it's valid JSON
  static client_saveSettings(settings: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      
      if (!fs.existsSync(CONFIG_DIR)) {
        const err = await makeConfigDir();
        if (err) {
          return reject(`Failed to create config dir: ${err}`);
        }
      }

      const fileOptions = {
        mode: 0o600,
        encoding: 'utf-8',
      };
      try {
        JSON.parse(settings); // Just ensure it's valid JSON
        fs.writeFile(SETTINGS_PATH, settings, fileOptions, async (err) => {
          if (err) {
            reject(err);
          } else {
            const updated = await this.client_getSettings();
            resolve(updated);
          }
        });
      } catch (err) {
        reject(err);
      }
    });
  }

  static client_exit() {
    process.on('unhandledRejection', () => { }); // STFU Node
    process.exit(0);
  }

  static config_list(): Promise<string> {
    return new Promise((resolve) => {
      fs.readdir(CONFIG_DIR, (_, items) => {
        if (!fs.existsSync(CONFIG_DIR) || items === undefined) {
          return resolve(JSON.stringify([]));
        }
        const configs: RPCConfig[] = [];
        for (let index = 0; index < items.length; ++index) {
          const filePath = path.join(CONFIG_DIR, items[index]);
          if (fs.existsSync(filePath) && !fs.lstatSync(filePath).isDirectory()) {
            const fileData = fs.readFileSync(filePath);
            configs.push(JSON.parse(fileData.toString('utf8')));
          }
        }
        resolve(JSON.stringify(configs));
      });
    });
  }

  @jsonSchema({
    "properties": {
      "configs": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "operator": {"type": "string", "minLength": 1},
            "lhost": {"type": "string", "minLength": 1},
            "lport": {"type": "number"},
            "ca_certificate": {"type": "string", "minLength": 1},
            "certificate": {"type": "string", "minLength": 1},
            "private_key": {"type": "string", "minLength": 1},
          }
        },
      },
    },
    "required": ["configs"]
  })
  static async config_save(req: string): Promise<string> {
    
    const configs: RPCConfig[] = JSON.parse(req).configs;
    if (!fs.existsSync(CONFIG_DIR)) {
      const err = await makeConfigDir();
      if (err) {
        return Promise.reject(`Failed to create config dir: ${err}`);
      }
    }
    const fileOptions = {
      mode: 0o600,
      encoding: 'utf-8',
    };

    await Promise.all(configs.map((config) => { 
      return new Promise((resolve) => {
        const fileName: string = uuid.v4();
        const data = JSON.stringify(config);
        fs.writeFile(path.join(CONFIG_DIR, fileName), data, fileOptions, (err) => {
          if (err) {
            console.error(err);
          }
          resolve();
        });
      });
    }));

    return IPCHandlers.config_list();
  }

  static async rpc_request(data: string): Promise<string> {
    const reqEnvelope: Envelope = Envelope.deserializeBinary(decodeRequest(data));
    const respEnvelope = await rpc.request(reqEnvelope);
    return base64.encode(respEnvelope.getData_asU8());
  }

  static async rpc_send(data: string): Promise<string> {
    const envelope: Envelope = Envelope.deserializeBinary(decodeRequest(data));
    rpc.sendEnvelope(envelope);
    return JSON.stringify({ success: true });
  }

}

async function dispatchIPC(method: string, data: string): Promise<Object | null> {
  console.log(`IPC Dispatch: ${method}`);

  // IPC handlers must start with "namespace_" this helps ensure we do not inadvertently
  // expose methods that we don't want exposed to the sandboxed code.
  if (['client_', 'config_', 'rpc_'].some(prefix => method.startsWith(prefix))) {
    if (typeof IPCHandlers[method] === 'function') {
      const result: string = await IPCHandlers[method](data);
      return result;
    } else {
      return Promise.reject(`No handler for method: ${method}`);
    }
  } else {
    return Promise.reject(`Invalid method handler namespace for "${method}"`);
  }
}

export function startIPCHandlers(window: BrowserWindow) {

  ipcMain.on('ipc', async (event: IpcMainEvent, msg: IPCMessage) => {
    dispatchIPC(msg.method, msg.data).then((result: string) => {
      if (msg.id !== 0) {
        event.sender.send('ipc', {
          id: msg.id,
          type: 'response',
          method: 'success',
          data: result
        });
      }
    }).catch((err) => {
      console.error(`[startIPCHandlers] ${err}`);
      if (msg.id !== 0) {
        event.sender.send('ipc', {
          id: msg.id,
          type: 'response',
          method: 'error',
          data: err.toString()
        });
      }
    });
  });

  // This one doesn't have an event argument for some reason ...
  ipcMain.on('push', async (_: IpcMainEvent, data: string) => {
    window.webContents.send('ipc', {
      id: 0,
      type: 'push',
      method: '',
      data: data
    });
  });

}
