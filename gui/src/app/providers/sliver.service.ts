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
import { BehaviorSubject } from 'rxjs';
import { RPCConfig } from '../../../rpc';


interface RPCResponse {
  pb: string;
}

@Injectable({
  providedIn: 'root'
})
export class SliverService {

  constructor(private _ipc: IPCService) { }

  // Holy shit, FUCK JAVASCRIPT
  decodeResp(response: RPCResponse): Uint8Array {
    const byteCharacters = atob(response.pb);
    const byteNumbers = new Array(byteCharacters.length);
    for (let index = 0; index < byteCharacters.length; index++) {
        byteNumbers[index] = byteCharacters.charCodeAt(index);
    }
    return new Uint8Array(byteNumbers);
  }

  async sessions(): Promise<clientpb.Sessions> {
    return new Promise(async (resolve, reject) => {
      try {
        const resp: RPCResponse = await this._ipc.request('rpc_sessions', '');
        resolve(clientpb.Sessions.deserializeBinary(this.decodeResp(resp)));
      } catch (err) {
        reject(err);
      }
    });
  }

}
