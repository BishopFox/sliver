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

IPC methods used to list/load RPCConfigs

*/

import * as fs from 'fs';
import * as path from 'path';
import { homedir } from 'os';
import { RPCConfig } from '../rpc';

const CONFIG_DIR = path.join(homedir(), '.sliver-client', 'configs');


export class ConfigHandlers {

  static config_list(): Promise<RPCConfig[]> {
    return new Promise((resolve) => {
      fs.readdir(CONFIG_DIR, (_, items) => {
        if (!fs.existsSync(CONFIG_DIR)) {
          resolve([]);
        }
        const configs: RPCConfig[] = [];
        for (let index = 0; index < items.length;  ++index) {
          const filePath = path.join(CONFIG_DIR, items[index]);
          if (fs.existsSync(filePath) && !fs.lstatSync(filePath).isDirectory()) {
            const fileData = fs.readFileSync(filePath);
            configs.push(JSON.parse(fileData.toString('utf8')));
          }
        }
        resolve(configs);
      });
    });
  }

  static configByLHost(lhost: string): Promise<RPCConfig> {
    return new Promise(async (resolve, reject) => {
      const configs = await this.config_list();
      for (let index = 0; index < configs.length; ++index) {
        if (configs[index].lhost === lhost) {
          resolve(configs[index]);
          return;
        }
      }
      reject('Config not found');
    });
  }

}
