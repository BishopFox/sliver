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
import { ServerHandlers } from './server.handlers';


let rpc: RPCClient;

// IPC Methods used to start/interact with the RPCClient
class RPCClientHandlers {

  static rpc_start(config: RPCConfig): Promise<any> {
    return new Promise(async (resolve) => {
      console.log(`Connecting to ${config.lhost}:${config.lport} ...`);
      rpc = new RPCClient(config);
      await rpc.connect();
      console.log('Connection successful');
      resolve(null);
    });
  }

  static rpc_activeConfig(): Promise<RPCConfig|null> {
    return new Promise((resolve) => {
      resolve(rpc ? rpc.config : null);
    });
  }

}

function dispatchIPC(method: string, data: string): Promise<Object|null> {
  return new Promise(async (resolve, reject) => {

    console.log(`IPC Dispatch: ${method} - ${data}`);

    // IPC handlers must start with "namespace_" this helps ensure we do not inadvertently
    // expose methods that we don't want exposed to the sandboxed code.
    if (['rpc_', 'config_', 'server_'].some(prefix => method.startsWith(prefix))) {
      if (typeof RPCClientHandlers[method] === 'function') {
        const result: Object|null = await RPCClientHandlers[method](data);
        resolve(result);
        return;
      } else if (typeof ConfigHandlers[method] === 'function') {
        const result: Object|null = await ConfigHandlers[method](data);
        resolve(result);
        return;
      } else if (typeof ServerHandlers[method] === 'function') {
        if (rpc && rpc.isConnected) {
          const result: Object|null = await ServerHandlers[method](rpc, data);
          resolve(result);
          return;
        }
        reject('RPC client is not connected to server');
      }
      reject(`No handler for method: ${method}`);
    } else {
      reject('Invalid method handler namepsace');
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
    dispatchIPC(msg.method, msg.data).then((result) => {
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
