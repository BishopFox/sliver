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
import { IPCService } from './ipc.service';
import { RPCConfig } from '../../../rpc';


@Injectable({
  providedIn: 'root'
})
export class ConfigService {

  constructor(private _ipc: IPCService) { }

  async getActiveConfig(): Promise<RPCConfig> {
    return new Promise(async (resolve) => {
      this._ipc.request('client_activeConfig', null).then((config) => {
        resolve(config);
      }).catch(() => {
        resolve(null);
      });
    });
  }

  async setActiveConfig(config: RPCConfig) {
    return new Promise(async (resolve) => {
      const data = await this._ipc.request('client_start', config);
      resolve(data);
    });
  }

  async listConfigs(): Promise<RPCConfig[]> {
    return new Promise(async (resolve) => {
      const configs: RPCConfig[] = await this._ipc.request('config_list', null);
      resolve(configs);
    });
  }

}
