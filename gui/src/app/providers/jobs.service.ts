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
