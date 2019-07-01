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
import * as clientpb from '../../../rpc/pb/client_pb';
import * as sliverpb from '../../../rpc/pb/sliver_pb';

@Injectable()
export class SliverService {

  private ipc: IPCService;

  constructor(ipc: IPCService) {
    this.ipc = ipc;
  }

  async sessions(): Promise<clientpb.Sessions> {
    return new Promise(async (resolve, reject) => {
      try {
        const sessions: clientpb.Sessions = await this.ipc.request('sliver_sessions', '');
        resolve(sessions);
      } catch (err) {
        reject(err);
      }
    });
  }

}
