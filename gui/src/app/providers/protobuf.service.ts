import { Injectable } from '@angular/core';
import * as base64 from 'base64-arraybuffer';

import * as sliverpb from '../../../rpc/pb/sliver_pb';

@Injectable({
  providedIn: 'root'
})
export class ProtobufService {

  // PB/Envelope Decode
  decode(data: string): Uint8Array {
    const buf = base64.decode(data);
    return new Uint8Array(buf);
  }

  // PB/Envelope Encode
  encode(request: sliverpb.Envelope): string {
    return base64.encode(request.serializeBinary());
  }

}
