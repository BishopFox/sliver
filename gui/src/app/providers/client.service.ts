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
*/

import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

import { IPCService } from './ipc.service';
import { RPCConfig } from '../../../rpc';


@Injectable({
  providedIn: 'root'
})
export class ClientService {

  isConnected$: BehaviorSubject<boolean> = new BehaviorSubject(false);

  constructor(private _ipc: IPCService) { }

  async getActiveConfig(): Promise<RPCConfig> {
    return new Promise(async (resolve) => {
      this._ipc.request('client_activeConfig', '').then((rawConfig) => {
        resolve(JSON.parse(rawConfig));
      }).catch(() => {
        resolve(null);
      });
    });
  }

  async setActiveConfig(config: RPCConfig) {
    return new Promise(async (resolve) => {
      const data = await this._ipc.request('client_start', JSON.stringify(config));
      this.isConnected$.next(true);
      resolve(data);
    });
  }

  async listConfigs(): Promise<RPCConfig[]> {
    return new Promise(async (resolve) => {
      const resp: string = await this._ipc.request('config_list', '');
      const configs: RPCConfig[] = JSON.parse(resp);
      console.log(configs);
      console.log(typeof configs);
      resolve(configs);
    });
  }

  exit() {
    this._ipc.request('client_exit', '');
  }

}
