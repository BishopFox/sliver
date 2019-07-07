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

import { ipcMain } from 'electron';
import { homedir } from 'os';
import * as base64 from 'base64-arraybuffer';
import * as fs from 'fs';
import * as path from 'path';


import { RPCClient, RPCConfig } from '../rpc';
import { Envelope } from '../rpc/pb';


const CONFIG_DIR = path.join(homedir(), '.sliver-client', 'configs');
let rpc: RPCClient;

function encodeResponse(data: Uint8Array): string {
  return base64.encode(data);
}

function decodeRequest(data: string): Uint8Array {
  const buf = base64.decode(data);
  return new Uint8Array(buf);
}


// IPC Methods used to start/interact with the RPCClient
class RPCClientHandlers {

  static client_start(data: string): Promise<string> {
    return new Promise(async (resolve, reject) => {
      const config: RPCConfig = JSON.parse(data);
      rpc = new RPCClient(config);
      rpc.connect().then(() => {
        console.log('Connection successful');
        rpc.envelopeSubject$.subscribe((envelope) => {
          if (envelope.getId() === 0) {
            ipcMain.emit('push', encodeResponse(envelope.getData_asU8()));
          }
        });
        resolve('success');
      }).catch((err) => {
        reject(err);
      });
    });
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
      const request: Envelope = Envelope.deserializeBinary(decodeRequest(data));
      const respEnvelope = await rpc.request(request);
      resolve(encodeResponse(respEnvelope.getData_asU8()));
    });
  }

  static client_exit() {
    process.on('unhandledRejection', () => {}); // STFU Node
    process.exit(0);
  }

}

function dispatchIPC(method: string, data: string): Promise<Object|null> {
  return new Promise(async (resolve, reject) => {

    console.log(`IPC Dispatch: ${method} - ${data}`);

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
      event.sender.send('ipc', {
        id: msg.id,
        type: 'response',
        method: 'success',
        data: result
      });
    }).catch((err) => {
      event.sender.send('ipc', {
        id: msg.id,
        type: 'response',
        method: 'error',
        data: err.toString()
      });
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
