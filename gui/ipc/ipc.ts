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
import { RPCClient, RPCConfig } from '../rpc';
import { ConfigHandlers } from './config.handlers';
import { RPCHandlers } from './rpc.handlers';


let rpc: RPCClient;

// IPC Methods used to start/interact with the RPCClient
class RPCClientHandlers {

  static client_start(data: string): Promise<any> {
    return new Promise(async (resolve) => {
      const config: RPCConfig = JSON.parse(data);
      rpc = new RPCClient(config);
      await rpc.connect();
      console.log('Connection successful');
      resolve(null);
    });
  }

  static client_activeConfig(): Promise<RPCConfig|null> {
    return new Promise((resolve) => {
      resolve(rpc ? rpc.config : null);
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
        const result: Object|null = await RPCClientHandlers[method](data);
        resolve(result);
        return;
      } else if (typeof ConfigHandlers[method] === 'function') {
        const result: Object|null = await ConfigHandlers[method](data);
        resolve(result);
        return;
      } else if (typeof RPCHandlers[method] === 'function') {
        if (rpc && rpc.isConnected) {
          const result: Object|null = await RPCHandlers[method](rpc, data);
          resolve(result);
          return;
        }
        reject('RPC client is not connected to server');
      }
      reject(`No handler for method: ${method}`);
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

export function startIPCHandlers() {
  ipcMain.on('ipc', async (event: any, msg: IPCMessage) => {
    dispatchIPC(msg.method, msg.data).then((result: Object) => {
      event.sender.send('ipc', {
        id: msg.id,
        type: 'response',
        method: 'success',
        data: JSON.stringify(result)
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
}
