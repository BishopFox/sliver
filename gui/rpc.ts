import { Subject, Observable, Observer } from 'rxjs';
import { TLSSocket, ConnectionOptions, TlsOptions, connect } from 'tls';
import * as pb from './pb/sliver_pb';
import * as msg from './pb/constants';

export interface RPCConfig {
  operator: string;
  lhost: string;
  lport: number;
  ca_certificate: string;
  certificate: string;
  private_key: string;
}

export class RPCClient {

  private config: RPCConfig;
  private socket: TLSSocket;
  private recvBuffer: Buffer;
  private isConnected = false;

  constructor(config: RPCConfig) {
    this.config = config;
  }

  // This method returns a Subject that shits out
  // or takes in pb.Envelopes and abstracts the byte
  // non-sense for your.
  async connect(): Promise<Subject<pb.Envelope>> {
    return new Promise(async (resolve, reject) => {
      if (this.isConnected) {
        reject('Already connected to rpc server');
      }

      const tlsSubject = await this.tlsConnect();
      this.isConnected = true;

      const envelopeObservable = Observable.create((obs: Observer<pb.Envelope>) => {
        this.recvBuffer = Buffer.alloc(0);
        tlsSubject.subscribe((recvData: Buffer) => {
          this.recvBuffer = Buffer.concat([this.recvBuffer, recvData]);
          if (4 <= this.recvBuffer.length) {
            const readSize = new Int32Array(this.recvBuffer.slice(0, 4))[0];
            console.log(`Recv msg length: ${readSize}`);
            if (readSize <= 4 + this.recvBuffer.length) {
              const bytes = this.recvBuffer.slice(4, 4 + readSize);
              const envelope = pb.Envelope.deserializeBinary(bytes);
              obs.next(envelope);
            }
          }
        });
      });

      const envelopeObserver = {
        next: (envelope: pb.Envelope) => {
          const dataBuffer = Buffer.from(envelope.serializeBinary());
          const sizeBuffer = this.toBytesUint32(dataBuffer.length);
          tlsSubject.next(Buffer.concat([sizeBuffer, dataBuffer]));
        }
      };

      resolve(Subject.create(envelopeObserver, envelopeObservable));
    });
  }

  private toBytesUint32(num: number): Buffer {
    const arr = new ArrayBuffer(4); // an Int32 takes 4 bytes
    const view = new DataView(arr);
    view.setUint32(0, num, false); // byteOffset = 0; litteEndian = false
    return Buffer.from(arr);
  }

  get tlsOptions(): ConnectionOptions {
    return {
      ca: this.config.ca_certificate,
      key: this.config.private_key,
      cert: this.config.certificate,
      host: this.config.lhost,
      port: this.config.lport,
      rejectUnauthorized: true,

      // This should ONLY skip verifying the hostname matches the cerftificate:
      // https://nodejs.org/api/tls.html#tls_tls_checkserveridentity_hostname_cert
      checkServerIdentity: () => undefined,
    };
  }

  // This is somehow the "clean" way to do this shit...
  // tlsConnect returns a Subject that shits out Buffers
  // or takes in Buffers of an interminate size as they come
  private tlsConnect(): Promise<Subject<Buffer>> {
    return new Promise((resolve, reject) => {

      console.log(`Connecting to ${this.config.lhost}:${this.config.lport} ...`);

      // Conenct to the server
      this.socket = connect(this.tlsOptions);

      // This event fires after the tls handshake, but we need to check `socket.authorized`
      this.socket.on('secureConnect', () => {
        console.log('RPC client connected', this.socket.authorized ? 'authorized' : 'unauthorized');
        if (this.socket.authorized === true) {

          const socketObservable = Observable.create((obs: Observer<Buffer>) => {
            this.socket.on('data', obs.next.bind(obs));    // Bind observable's .next() to 'data' event
            this.socket.on('close', obs.error.bind(obs));  // same with close/error
          });

          const socketObserver = {
            next: (data: Buffer) => {
              this.socket.write(data); // Bind subject's .next() to socket's .write()
            }
          };

          resolve(Subject.create(socketObserver, socketObservable));
        } else {
          reject('Unauthorized connection');
        }
      });
    });
  }

}
