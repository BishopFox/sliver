import { Subject, Observable, Observer } from 'rxjs';
import { TLSSocket, ConnectionOptions, TlsOptions, connect } from 'tls';


export interface RPCConfig {
  operator: string;
  lhost: string;
  lport: number;
  ca_certificate: string;
  certificate: string;
  private_key: string;
}

export interface Envelope {
  msgType: number;
  data: string;
}

export class RPCClient {

  private config: RPCConfig;
  private socket: TLSSocket;

  constructor(config: RPCConfig) {
    this.config = config;
  }

  async connect() {
    const tlsSubject = await this.tlsConnect();
    tlsSubject.subscribe((recvData: Buffer) => {
      console.log(recvData);
    });
    console.log('Sending data ...');
    const sendData = new Buffer('test');
    tlsSubject.next(sendData);
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
  private tlsConnect(): Promise<Subject<Buffer>> {
    return new Promise((resolve, reject) => {

      console.log(`Connecting to ${this.config.lhost}:${this.config.lport} ...`);

      // Conenct to the server
      this.socket = connect(this.tlsOptions);

      // This event fires after the tls handshake, but we need to check `socket.authorized`
      this.socket.on('secureConnect', () => {
        console.log('RPC client connected', this.socket.authorized ? 'authorized' : 'unauthorized');
        if (this.socket.authorized === true) {

          const observable = Observable.create((obs: Observer<Buffer>) => {
            this.socket.on('data', obs.next.bind(obs));    // Bind observable's .next() to 'data' event
            this.socket.on('close', obs.error.bind(obs));  // same with close/error
          });

          const observer = {
            next: (data: Buffer) => {
              this.socket.write(data); // Bind subject's .next() to socket's .write()
            }
          };

          resolve(Subject.create(observer, observable));
        } else {
          reject('Unauthorized connection');
        }
      });
    });
  }

}
