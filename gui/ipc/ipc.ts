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

import { ipcMain, dialog, FileFilter } from 'electron';
import { homedir } from 'os';
import * as base64 from 'base64-arraybuffer';
import * as fs from 'fs';
import * as path from 'path';

import { RPCClient, RPCConfig } from '../rpc';
import { Envelope } from '../rpc/pb';


const CLIENT_DIR = path.join(homedir(), '.sliver-client');
const CONFIG_DIR = path.join(CLIENT_DIR, 'configs');
const SETTINGS_FILEPATH = path.join(CLIENT_DIR, 'gui-settings.json');


let rpc: RPCClient;


function decodeRequest(data: string): Uint8Array {
  const buf = base64.decode(data);
  return new Uint8Array(buf);
}

interface SaveFileReq {
  title: string;
  message: string;
  filename: string;
  data: string;
}

interface ReadFileReq {
  title: string;
  message: string;
  openDirectory: boolean;
  multiSelections: boolean;
  filters: FileFilter[] | null; // { filters: [ { name: 'Custom File Type', extensions: ['as'] } ] }
}

// IPC Methods used to start/interact with the RPCClient
class RPCClientHandlers {

  static client_start(req: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      const config: RPCConfig = JSON.parse(req);
      rpc = new RPCClient(config);
      rpc.connect().then(() => {
        console.log('Connection successful');
        rpc.envelopeSubject$.subscribe((envelope) => {
          if (envelope.getId() === 0) {
            ipcMain.emit('push', envelope.getData_asB64());
          }
        });
        resolve('success');
      }).catch((err) => {
        reject(err);
      });
    });
  }

  static client_readFile(req: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      try {
        const readFileReq: ReadFileReq = JSON.parse(req);
        const dialogOptions = {
          title: readFileReq.title,
          message: readFileReq.message,
          openDirectory: readFileReq.openDirectory,
          multiSelections: readFileReq.multiSelections
        };
        dialog.showOpenDialog(null, dialogOptions, (filePaths) => {
          if (filePaths) {

            // Well this is kinda nasty and nested, but basically
            // we're just doing 'n' async files reads and putting
            // them all into `files` but we need to wait for all
            // 'n' reads to complete before resolve'ing the promise
            const files = [];
            Promise.all(filePaths.map((filePath) => {
              return new Promise(async (resolveRead) => {
                fs.readFile(filePath, (err, data) => {
                  files.push({
                    filePath: filePath,
                    error: err.toString(),
                    data: data ? base64.encode(data) : null
                  });
                  resolveRead();
                });
              });
            })).then(() => {
              resolve(JSON.stringify({files: files}));
            });

          } else {
            resolve('');
          }
        });
      } catch (err) {
        reject(err);
      }
    });
  }

  // For now all files are just saved to the Downloads folder,
  // which should exist on all supported platforms.
  static client_saveFile(req: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      const saveFileReq: SaveFileReq = JSON.parse(req);
      const dialogOptions = {
        title: saveFileReq.title,
        message: saveFileReq.message,
        defaultPath: path.join(homedir(), 'Downloads', path.basename(saveFileReq.filename)),
      };
      dialog.showSaveDialog(null, dialogOptions, (filename) => {
        console.log(`[save file] ${filename}`);
        if (filename) {
          const fileOptions = {
            mode: 0o644,
            encoding: 'binary',
          };
          const data = Buffer.from(base64.decode(saveFileReq.data));
          fs.writeFile(filename, data, fileOptions, (err) => {
            if (err) {
              reject(err);
            } else {
              resolve(JSON.stringify({filename: filename}));
            }
          });
        } else {
          resolve(''); // User hit 'cancel'
        }
      });
    });
  }

  static client_getSettings(): Promise<string> {
    return new Promise((resolve, reject) => {
      try {
        if (!fs.existsSync(SETTINGS_FILEPATH)) {
          resolve('{}');
        }
        fs.readFile(SETTINGS_FILEPATH, 'utf-8', (err, data) => {
          if (err) {
            reject(err);
          }
          JSON.parse(data);
          resolve(data);
        });
      } catch (err) {
        reject(err);
      }
    });
  }

  static client_saveSettings(settings: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      const options = {
        mode: 0o644,
        encoding: 'utf-8',
      };
      try {
        JSON.parse(settings);
        fs.writeFile(SETTINGS_FILEPATH, settings, options, async (err) => {
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
    process.on('unhandledRejection', () => {}); // STFU Node
    process.exit(0);
  }

  static config_list(): Promise<string> {
    return new Promise((resolve) => {
      fs.readdir(CONFIG_DIR, (_, items) => {
        if (!fs.existsSync(CONFIG_DIR)) {
          resolve(JSON.stringify([]));
        }
        const configs: RPCConfig[] = [];
        for (let index = 0; index < items.length;  ++index) {
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

  static client_activeConfig(): Promise<string> {
    return new Promise((resolve) => {
      resolve(rpc ? JSON.stringify(rpc.config) : '');
    });
  }

  static rpc_request(data: string): Promise<string> {
    return new Promise(async (resolve) => {
      const reqEnvelope: Envelope = Envelope.deserializeBinary(decodeRequest(data));
      const respEnvelope = await rpc.request(reqEnvelope);
      resolve(base64.encode(respEnvelope.getData_asU8()));
    });
  }

  static rpc_send(data: string): Promise<string> {
    return new Promise((resolve, reject) => {
      try {
        const envelope: Envelope = Envelope.deserializeBinary(decodeRequest(data));
        rpc.sendEnvelope(envelope);
        resolve(JSON.stringify({sucess: true}));
      } catch (err) {
        reject(err);
      }
    });
  }

}

function dispatchIPC(method: string, data: string): Promise<Object|null> {
  return new Promise(async (resolve, reject) => {

    console.log(`IPC Dispatch: ${method}`);

    // IPC handlers must start with "namespace_" this helps ensure we do not inadvertently
    // expose methods that we don't want exposed to the sandboxed code.
    if (['client_', 'config_', 'rpc_'].some(prefix => method.startsWith(prefix))) {
      if (typeof RPCClientHandlers[method] === 'function') {
        try {
          const result: string = await RPCClientHandlers[method](data);
          resolve(result);
        } catch (err) {
          reject(err);
        }
      } else {
        reject(`No handler for method: ${method}`);
      }
    } else {
      reject(`Invalid method handler namepsace for "${method}"`);
    }
  });
}

interface IPCMessage {
  id: number;
  type: string;
  method: string; // Identifies the target method and in the response if the method call was a success/error
  data: string;
}

export function startIPCHandlers(window) {

  ipcMain.on('ipc', async (event: any, msg: IPCMessage) => {
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
      if (msg.id !== 0) {
        event.sender.send('ipc', {
          id: msg.id,
          type: 'response',
          method: 'error',
          data: err.toString()
        });
      } else {
        console.error(err);
      }
    });
  });

  // This one doesn't have an event argument for some reason ...
  ipcMain.on('push', async (data: string) => {
    window.webContents.send('ipc', {
      id: 0,
      type: 'push',
      method: '',
      data: data
    });
  });

}
