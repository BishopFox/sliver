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
import { Subject } from 'rxjs';
import * as pb from '../../../rpc';


interface IPCMessage {
  id: number;
  type: string;
  method: string;
  data: string;
}

@Injectable({
  providedIn: 'root'
})
export class IPCService {

  private ipcMessageSubject: Subject<IPCMessage>;

  constructor() {
    this.ipcMessageSubject = new Subject<IPCMessage>();
    window.addEventListener('message', (event) => {
      console.log('web ipc recv:');
      console.log(event);
      try {
        const msg: IPCMessage = JSON.parse(event.data);
        if (msg.type === 'response') {
          this.ipcMessageSubject.next(msg);
        }
      } catch (err) {
        console.error(err);
      }
    });
  }

  async request(method: string, data: Object|null): Promise<any> {
    return new Promise((resolve) => {
      const msgId = this.randomId();
      const subscription = this.ipcMessageSubject.subscribe((msg: IPCMessage) => {
        if (msg.id === msgId) {
          subscription.unsubscribe();
          resolve(JSON.parse(msg.data));
        }
      });
      window.postMessage({
        id: msgId,
        type: 'request',
        method: method,
        data: data ? JSON.stringify(data) : '',
      }, '*');
    });
  }

  private randomId(): number {
    const buf = new Uint32Array(1);
    window.crypto.getRandomValues(buf);
    const bufView = new DataView(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
    return bufView.getUint32(0, true);
  }

}
