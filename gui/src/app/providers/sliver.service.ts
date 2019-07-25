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

This service is responsible for all of the Sliver Server interactions that
use protobuf.

*/

import { Injectable } from '@angular/core';
import { IPCService } from './ipc.service';
import { ProtobufService } from './protobuf.service';
import * as pb from '@rpc/pb';


@Injectable({
  providedIn: 'root'
})
export class SliverService extends ProtobufService {

  constructor(private _ipc: IPCService) {
    super();
  }

  async sessions(): Promise<pb.Sessions> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.ClientPB.MsgSessions);
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Sessions.deserializeBinary(this.decode(resp));
  }

  async sessionById(id: number): Promise<pb.Sliver> {
    const sessions = await this.sessions();
    const slivers = sessions.getSliversList();
    for (let index = 0; index < slivers.length; ++index) {
      if (slivers[index].getId() === id) {
        return slivers[index];
      }
    }
    return Promise.reject(`No session with id '${id}'`);
  }

  async sliverBuilds(): Promise<pb.SliverBuilds> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.ClientPB.MsgListSliverBuilds);
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.SliverBuilds.deserializeBinary(this.decode(resp));
  }

  async canaries(): Promise<pb.Canaries> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.ClientPB.MsgListCanaries);
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Canaries.deserializeBinary(this.decode(resp));
  }

  async generate(config: pb.SliverConfig): Promise<pb.Generate> {
    const reqEnvelope = new pb.Envelope();
    const generateReq = new pb.GenerateReq();
    generateReq.setConfig(config);
    reqEnvelope.setType(pb.ClientPB.MsgGenerate);
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Generate.deserializeBinary(this.decode(resp));
  }

  async regenerate(name: string): Promise<pb.Regenerate> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.ClientPB.MsgRegenerate);
    const regenReq = new pb.Regenerate();
    regenReq.setSlivername(name);
    reqEnvelope.setData(regenReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Regenerate.deserializeBinary(this.decode(resp));
  }

  async ps(sliverId: number): Promise<pb.Ps> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgPsReq);
    const psReq = new pb.PsReq();
    psReq.setSliverid(sliverId);
    reqEnvelope.setData(psReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Ps.deserializeBinary(this.decode(resp));
  }

  async ls(sliverId: number, targetDir: string): Promise<pb.Ls> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgLsReq);
    const lsReq = new pb.LsReq();
    lsReq.setSliverid(sliverId);
    lsReq.setPath(targetDir);
    reqEnvelope.setData(lsReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Ls.deserializeBinary(this.decode(resp));
  }

  async cd(sliverId: number, targetDir: string): Promise<pb.Pwd> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgCdReq);
    const cdReq = new pb.CdReq();
    cdReq.setSliverid(sliverId);
    cdReq.setPath(targetDir);
    reqEnvelope.setData(cdReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Pwd.deserializeBinary(this.decode(resp));
  }

  async rm(sliverId: number, target: string): Promise<pb.Rm> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgRmReq);
    const rmReq = new pb.RmReq();
    rmReq.setSliverid(sliverId);
    rmReq.setPath(target);
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Rm.deserializeBinary(this.decode(resp));
  }

  async mkdir(sliverId: number, targetDir: string): Promise<pb.Mkdir> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgMkdirReq);
    const mkdirReq = new pb.MkdirReq();
    mkdirReq.setSliverid(sliverId);
    mkdirReq.setPath(targetDir);
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Mkdir.deserializeBinary(this.decode(resp));
  }

  async download(sliverId: number, targetFile: string): Promise<pb.Download> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgDownloadReq);
    const downloadReq = new pb.DownloadReq();
    downloadReq.setSliverid(sliverId);
    downloadReq.setPath(targetFile);
    reqEnvelope.setData(downloadReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Download.deserializeBinary(this.decode(resp));
  }

  async upload(sliverId: number, data: Uint8Array, encoder: string, dst: string): Promise<pb.Upload> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgUploadReq);
    const uploadReq = new pb.UploadReq();
    uploadReq.setSliverid(sliverId);
    uploadReq.setData(data);
    uploadReq.setEncoder(encoder);
    uploadReq.setPath(dst);
    reqEnvelope.setData(uploadReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Upload.deserializeBinary(this.decode(resp));
  }

  async ifconfig(sliverId: number): Promise<pb.Ifconfig> {
    const reqEnvelope = new pb.Envelope();
    reqEnvelope.setType(pb.SliverPB.MsgIfconfigReq);
    const ifconfigReq = new pb.IfconfigReq();
    ifconfigReq.setSliverid(sliverId);
    reqEnvelope.setData(ifconfigReq.serializeBinary());
    const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
    return pb.Ifconfig.deserializeBinary(this.decode(resp));
  }

}
