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

This service is talks to the mTLS client and manages configs/etc.

*/

import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';
import * as base64 from 'base64-arraybuffer';

import { IPCService } from './ipc.service';
import { RPCConfig } from '@rpc/rpc';
import { FileFilter } from 'electron';


export interface SaveFileReq {
  title: string;
  message: string;
  filename: string;
  data: string;
}

export interface ReadFile {
  filePath: string|null;
  error: string|null;
  data: string|null;
}

export interface ReadFiles {
  files: Array<ReadFile>;
}

export interface Settings {

  preferredServer: string;

  notifications: {
    sessionOpened: boolean;
    sessionClosed: boolean;

    jobStopped: boolean;

    playerJoined: boolean;
    playerLeft: boolean;
  };

}


@Injectable({
  providedIn: 'root'
})
export class ClientService {

  isConnected$: BehaviorSubject<boolean> = new BehaviorSubject(false);

  constructor(private _ipc: IPCService) { }

  async getActiveConfig(): Promise<RPCConfig> {
    const rawConfig = await this._ipc.request('client_activeConfig');
    return rawConfig ? JSON.parse(rawConfig) : null;
  }

  async setActiveConfig(config: RPCConfig): Promise<string> {
    const data = await this._ipc.request('client_start', JSON.stringify(config));
    this.isConnected$.next(true);
    return data;
  }

  async getSettings(): Promise<Settings> {
    const data = await this._ipc.request('client_getSettings');
    return data ? JSON.parse(data) : {};
  }

  async setSettings(settings: Settings): Promise<Settings> {
    await this._ipc.request('client_setSettings', JSON.stringify(settings));
    return this.getSettings();
  }

  async listConfigs(): Promise<RPCConfig[]> {
    const resp = await this._ipc.request('config_list');
    const configs: RPCConfig[] = JSON.parse(resp);
    return configs;
  }

  async saveConfigs(configs: RPCConfig[]): Promise<string> {
    return this._ipc.request('config_save', JSON.stringify({ configs: configs })); 
  }

  async saveFile(title: string, message: string, filename: string, data: Uint8Array): Promise<string> {
    return await this._ipc.request('client_saveFile', JSON.stringify({
      title: title,
      message: message,
      filename: filename,
      data: base64.encode(data),
    }));
  }

  async readFile(title: string, message: string, openDirectory?: boolean,
                 multiSelection?: boolean, filter?: FileFilter[]): Promise<ReadFiles|null> {
    const resp = await this._ipc.request('client_readFile', JSON.stringify({
      title: title,
      message: message,
      openDirectory: openDirectory !== undefined ? openDirectory : false,
      multiSelection: multiSelection !== undefined ? multiSelection : false,
      filter: filter !== undefined ? filter : [{
        name: 'All Files', extensions: ['*']
      }],
    }));
    return resp ? JSON.parse(resp) : null;
  }

  exit() {
    this._ipc.request('client_exit', '');
  }

}
