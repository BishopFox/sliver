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
import { Subject, Observer } from 'rxjs';
import { IPCService } from './ipc.service';
import { ProtobufService } from './protobuf.service';
import * as pb from '../../../rpc/pb';


export interface Tunnel {
  id: number;
  sliverID: number;
  send: Observer<Buffer>;
  recv: Subject<Buffer>;
}


@Injectable({
  providedIn: 'root'
})
export class TunnelService extends ProtobufService {

  constructor(private _ipc: IPCService) {
    super();
  }

  createTunnel(sliverID: number): Promise<Tunnel> {
    return new Promise(async (resolve, reject) => {
      try {
        const envelope = new pb.Envelope();
        envelope.setType(pb.ClientPB.MsgTunnelCreate);
        const tunReq = new pb.TunnelCreateReq();
        tunReq.setSliverid(sliverID);
        envelope.setData(tunReq.serializeBinary());
        const resp = await this._ipc.request('rpc_request', this.encode(envelope));
        const tun: pb.TunnelCreate = pb.TunnelCreate.deserializeBinary(this.decode(resp));

        const recv$ = new Subject<Buffer>();
        const tunSub = this._ipc.ipcTunnelData$.subscribe((tunData) => {
          if (tunData.getTunnelid() === tun.getTunnelid() && tunData.getSliverid() === tun.getSliverid()) {
            const data = tunData.getData_asU8();
            console.log(`[tunnel] Recv ${data.length} byte(s) on Tunnel ${tun.getTunnelid()}`);
            recv$.next(Buffer.from(data));
          }
        });

        const sendObs: Observer<Buffer> = {
          next: (data: Buffer) => {
            console.log(`[tunnel] Send ${data.length} byte(s) on Tunnel ${tun.getTunnelid()}`);
            const sendEnvelope = new pb.Envelope();
            sendEnvelope.setType(pb.SliverPB.MsgTunnelData);
            const sendTunData = new pb.TunnelData();
            sendTunData.setTunnelid(tun.getTunnelid());
            sendTunData.setSliverid(tun.getSliverid());
            sendTunData.setData(data);
            this._ipc.sendEnvelope(sendEnvelope);
          },
          complete: () => {
            tunSub.unsubscribe();
            recv$.complete();
          },
          error: console.error,
        };

        resolve({
          id: tun.getTunnelid(),
          sliverID: tun.getSliverid(),
          send: sendObs,
          recv: recv$,
        });
      } catch (err) {
        reject(err);
      }
    });
  }

}
