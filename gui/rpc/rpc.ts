import { Subject, Observable, Observer } from 'rxjs';
import tls from 'tls';

export class MTLSService {

    private server: string;
    private port: number;
    private ca: string;
    private client_certificate: string;
    private client_key: string;

    constructor(server: string, port: number, ca: string, certificate: string, key: string) {
        this.server = server;
        this.port = port;
        this.ca = ca;
        this.client_key = key;
        this.client_certificate = certificate;
    }

    get options(): object {
        return {
            ca: this.ca,
            key: this.client_key,
            cert: this.client_certificate,
            host: this.server,
            port: this.port,
            rejectUnauthorized: true,
            requestCert: true
        };
    }

    connect(): tls.TLSSocket {
        const socket = tls.connect(this.options, () => {
            console.log('client connected', socket.authorized ? 'authorized' : 'unauthorized');
            process.stdin.pipe(socket);
            process.stdin.resume();
        });
        socket.setEncoding('utf8');
        return socket;
    }

}
