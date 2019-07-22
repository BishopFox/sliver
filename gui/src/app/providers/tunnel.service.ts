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
  send: Observer<Uint8Array>;
  recv: Subject<Uint8Array>;
}


@Injectable({
  providedIn: 'root'
})
export class TunnelService extends ProtobufService {

  constructor(private _ipc: IPCService) {
    super();
  }

  createTunnel(sliverId: number, enablePty?: boolean): Promise<Tunnel> {
    return new Promise(async (resolve, reject) => {
      try {

        // Open Tunnel
        const tunReqEnvl = new pb.Envelope();
        tunReqEnvl.setType(pb.ClientPB.MsgTunnelCreate);
        const tunReq = new pb.TunnelCreateReq();
        tunReq.setSliverid(sliverId);
        tunReqEnvl.setData(tunReq.serializeBinary());
        const tunResp = await this._ipc.request('rpc_request', this.encode(tunReqEnvl));
        const tun: pb.TunnelCreate = pb.TunnelCreate.deserializeBinary(this.decode(tunResp));

        // Request a shell
        const shellReqEnvl = new pb.Envelope();
        shellReqEnvl.setType(pb.SliverPB.MsgShellReq);
        const shellReq = new pb.ShellReq();
        shellReq.setSliverid(tun.getSliverid());
        shellReq.setTunnelid(tun.getTunnelid());
        shellReq.setEnablepty(enablePty === undefined ? true : enablePty);
        shellReqEnvl.setData(shellReq.serializeBinary());
        const resp = await this._ipc.request('rpc_request', this.encode(shellReqEnvl));
        console.log(resp);

        // Setup duplex communication
        const recv$ = new Subject<Uint8Array>();
        const tunSub = this._ipc.ipcTunnelData$.subscribe((recvData) => {
          if (recvData.getTunnelid() === tun.getTunnelid() && recvData.getSliverid() === tun.getSliverid()) {
            const data = recvData.getData_asU8();
            console.log(`[tunnel] Recv ${data.length} byte(s) on Tunnel ${tun.getTunnelid()}`);
            recv$.next(data);
          }
        });

        const sendObs: Observer<Uint8Array> = {
          next: (data: Uint8Array) => {
            console.log(`[tunnel] Send ${data.length} byte(s) on Tunnel ${tun.getTunnelid()}`);
            const sendEnvl = new pb.Envelope();
            sendEnvl.setType(pb.SliverPB.MsgTunnelData);
            const sendData = new pb.TunnelData();
            sendData.setTunnelid(tun.getTunnelid());
            sendData.setSliverid(tun.getSliverid());
            sendData.setData(data);
            sendEnvl.setData(sendData.serializeBinary());
            this._ipc.sendEnvelope(sendEnvl);
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
