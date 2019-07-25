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

This service is the common carrier for all IPC messages.

*/

import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';
import * as pb from '@rpc/pb';
import { ProtobufService } from './protobuf.service';

interface IPCMessage {
  id: number;
  type: string;
  method: string;
  data: string;
}

@Injectable({
  providedIn: 'root'
})
export class IPCService extends ProtobufService {

  private _ipcResponse$ = new Subject<IPCMessage>();
  ipcEvent$ = new Subject<pb.Event>();
  ipcTunnelData$ = new Subject<pb.TunnelData>();
  ipcTunnelCtrl$ = new Subject<pb.TunnelClose>();

  constructor() {
    super();
    window.addEventListener('message', (ipcEvent) => {
      try {
        const msg: IPCMessage = JSON.parse(ipcEvent.data);
        if (msg.type === 'response') {
          this._ipcResponse$.next(msg);
        } else if (msg.type === 'push') {
          const envelope = pb.Envelope.deserializeBinary(this.decode(msg.data));
          switch (envelope.getType()) {
            case pb.ClientPB.MsgEvent:
              const event = pb.Event.deserializeBinary(envelope.getData_asU8());
              this.ipcEvent$.next(event);
              break;
            case pb.SliverPB.MsgTunnelData:
              const data = pb.TunnelData.deserializeBinary(envelope.getData_asU8());
              this.ipcTunnelData$.next(data);
              break;
            case pb.SliverPB.MsgTunnelClose:
              const tunCtrl = pb.TunnelClose.deserializeBinary(envelope.getData_asU8());
              this.ipcTunnelCtrl$.next(tunCtrl);
              break;
            default:
              console.error(`[IPCService] Unknown envelope type ${envelope.getType()}`);
          }
        }
      } catch (err) {
        console.error(`[IPCService] ${err}`);
      }
    });
  }

  request(method: string, data?: string): Promise<string> {
    return new Promise((resolve, reject) => {
      const msgId = this.randomId();
      const subscription = this._ipcResponse$.subscribe((msg: IPCMessage) => {
        if (msg.id === msgId) {
          subscription.unsubscribe();
          if (msg.method !== 'error') {
            resolve(msg.data);
          } else {
            reject(msg.data);
          }
        }
      });
      window.postMessage(JSON.stringify({
        id: msgId,
        type: 'request',
        method: method,
        data: data,
      }), '*');
    });
  }

  // Send envelope, don't wait for response
  sendEnvelope(envelope: pb.Envelope) {
    window.postMessage(JSON.stringify({
      id: 0,
      type: 'request',
      method: 'rpc_send',
      data: this.encode(envelope),
    }), '*');
  }

  private randomId(): number {
    const buf = new Uint32Array(1);
    window.crypto.getRandomValues(buf);
    const bufView = new DataView(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
    return bufView.getUint32(0, true) || 1; // In the unlikely event we get a 0 value, return 1 instead
  }

}
