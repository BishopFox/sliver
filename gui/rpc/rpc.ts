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

import { Subject, Observable, Observer } from 'rxjs';
import { TLSSocket, ConnectionOptions, connect } from 'tls';
import { randomBytes } from 'crypto';
import * as sliverpb from './pb/sliver_pb';


export interface RPCConfig {
  operator: string;
  lhost: string;
  lport: number;
  ca_certificate: string;
  certificate: string;
  private_key: string;
}

export class RPCClient {

  private _config: RPCConfig;
  private socket: TLSSocket;
  private tlsObserver: Observer<Buffer>;
  private tlsObservable: Observable<Buffer>;
  private recvBuffer: Buffer;
  private _isConnected = false;
  private readonly defaultTimeout = 30 * 1000000000; // 30 seconds (in nano)

  envelopeSubject$: Subject<sliverpb.Envelope>;

  constructor(config: RPCConfig) {
    this._config = config;
  }

  get config(): RPCConfig {
    return this._config;
  }

  get isConnected(): Boolean {
    return this._isConnected;
  }

  // Send an RPC request and get the mapped response
  request(reqEnvelope: sliverpb.Envelope): Promise<sliverpb.Envelope> {
    return new Promise((resolve, reject) => {
      if (!this.isConnected) {
        reject('No server connection');
      }
      const respId = this.randomId();
      reqEnvelope.setId(respId);
      const subscription = this.envelopeSubject$.subscribe((respEnvelope) => {
        console.log(`Recv Envelope with ID ${respEnvelope.getId()} (want ${respId})`);
        if (respEnvelope.getId() === respId) {
          subscription.unsubscribe();
          resolve(respEnvelope);
        }
      });
      this.sendEnvelope(reqEnvelope);
    });
  }

  // This method creates a Subject that shits out pb.Envelopes
  // and abstracts the byte non-sense for your.
  async connect(): Promise<any> {
    if (this.isConnected) {
      return Promise.reject('Already connected to rpc server');
    }
    await this.tlsConnect();
    this._isConnected = true;
    this.recvBuffer = Buffer.alloc(0);
    this.envelopeSubject$ = new Subject<sliverpb.Envelope>();
    this.tlsObservable.subscribe((data: Buffer) => {
      this.recvEnvelope(this.envelopeSubject$, data);
    });
  }

  sendEnvelope(envelope: sliverpb.Envelope) {
    if (!envelope.getTimeout()) {
      envelope.setTimeout(this.defaultTimeout);
    }
    const dataBuffer = Buffer.from(envelope.serializeBinary());
    const sizeBuffer = this.toBytesUint32(dataBuffer.length);
    console.log(`Sending msg (${envelope.getType()}): ${dataBuffer.length} bytes ...`);
    this.tlsObserver.next(Buffer.concat([sizeBuffer, dataBuffer]));
  }

  // This method parses out Envelopes from the recvBuffer stream, since we do not know
  // the length of the recvData ahead of time, we append it to a running buffer that we
  // then pull envelopes out of, we do this recursively since we may read two envelopes
  // worth of data in a single 'data' event. If we don't have enough data to parse out a
  // length and envelope then we just store it in .recvBuffer and wait for more data.
  private recvEnvelope(obs: Observer<sliverpb.Envelope>, recvData: Buffer) {
    this.recvBuffer = Buffer.concat([this.recvBuffer, recvData]);
    console.log(`Current recvBuffer is ${this.recvBuffer.length} bytes...`);
    if (4 < this.recvBuffer.length) {
      const lenBuf = this.recvBuffer.slice(0, 4);
      // Because the creators of this language are FUCKING IDIOTS, WHY THE FUCK IS THIS A THING
      // https://stackoverflow.com/questions/8609289/convert-a-binary-nodejs-buffer-to-javascript-arraybuffer/31394257#31394257
      const lenBufView = new DataView(lenBuf.buffer.slice(lenBuf.byteOffset, lenBuf.byteOffset + lenBuf.byteLength));
      const readSize = lenBufView.getUint32(0, true);  // byteOffset = 0; litteEndian = true
      console.log(`Recv msg length: ${readSize} bytes`);
      if (readSize <= 4 + this.recvBuffer.length) {
        console.log('Parsing envelope from recvBuffer');
        const bytes = this.recvBuffer.slice(4, 4 + readSize);
        const envelope = sliverpb.Envelope.deserializeBinary(bytes);
        console.log(`Deserialized msg Type = ${envelope.getType()}, ID = ${envelope.getId()}`);
        this.recvBuffer = Buffer.from(this.recvBuffer.slice(4 + readSize));
        obs.next(envelope);
        this.recvEnvelope(obs, Buffer.alloc(0)); // Recursively parse
      }
    }
  }

  // Convert a number to a 4 byte uint32 buffer
  private toBytesUint32(num: number): Buffer {
    const arr = new ArrayBuffer(4);
    const view = new DataView(arr);
    view.setUint32(0, num, true); // byteOffset = 0; litteEndian = true
    return Buffer.from(arr);
  }

  private randomId(): number {
    const buf = randomBytes(4);
    const bufView = new DataView(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
    return bufView.getUint32(0, true);
  }

  get tlsOptions(): ConnectionOptions {
    return {
      ca: this.config.ca_certificate,
      key: this.config.private_key,
      cert: this.config.certificate,
      host: this.config.lhost,
      port: this.config.lport,
      rejectUnauthorized: true,

      // This should ONLY skip verifying the hostname matches the certificate:
      // https://nodejs.org/api/tls.html#tls_tls_checkserveridentity_hostname_cert
      checkServerIdentity: () => undefined,
    };
  }

  // This is somehow the "clean" way to do this shit...
  // tlsConnect returns a Subject that shits out Buffers
  // or takes in Buffers of an indeterminate size as they come
  private tlsConnect() {
    return new Promise((resolve, reject) => {

      console.log(`Connecting to ${this.config.lhost}:${this.config.lport} ...`);

      this.socket = connect(this.tlsOptions);
      this.socket.on('error', (err) => {
        this.socket.destroy();
        reject(err);
      });

      // This event fires after the tls handshake, but we need to check `socket.authorized`
      this.socket.on('secureConnect', () => {
        console.log('RPC client connected', this.socket.authorized ? 'authorized' : 'unauthorized');
        if (this.socket.authorized === true) {

          this.socket.setNoDelay(true);

          this.tlsObservable = new Observable(producer => {
            this.socket.on('data', (readData: Buffer) => {
              console.log(`Socket read ${readData.length} bytes`);
              console.log(readData);
              producer.next(readData);
            });
            this.socket.on('close', producer.complete);
          });

          this.tlsObserver = {
            next: (data: Buffer) => {
              console.log(`Socket write ${data.length} bytes`);
              this.socket.write(data);
            },
            complete: () => {
              console.log('TLS Observer completed');
            },
            error: console.error,
          };

          resolve();
        } else {
          this.socket.destroy();
          reject('Unauthorized connection');
        }
      });
    });
  }

}
