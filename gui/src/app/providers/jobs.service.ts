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
import { ProtobufService } from './protobuf.service';
import * as pb from '../../../rpc/pb';


@Injectable({
  providedIn: 'root'
})
export class JobsService extends ProtobufService {

  constructor(private _ipc: IPCService) {
    super();
  }

  async jobs(): Promise<pb.Jobs> {
    return new Promise(async (resolve, reject) => {
      try {
        const reqEnvelope = new pb.Envelope();
        reqEnvelope.setType(pb.ClientPB.MsgJobs);
        const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
        resolve(pb.Jobs.deserializeBinary(this.decode(resp)));
      } catch (err) {
        reject(err);
      }
    });
  }
}
