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
import { IPCService } from './ipc.service';
import { ProtobufService } from './protobuf.service';
import * as pb from '@rpc/pb';


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

  async jobById(jobId: number): Promise<pb.Job> {
    return new Promise(async (resolve, reject) => {
      try {
        const jobs = await this.jobs();
        const activeJobs = jobs.getActiveList();
        for (let index = 0; index < activeJobs.length; ++index) {
          if (jobId === activeJobs[index].getId()) {
            resolve(activeJobs[index]);
            return;
          }
        }
        reject('Job not found');
      } catch (err) {
        reject(err);
      }
    });
  }

  async startMTLSListener(lport: number): Promise<pb.Job> {
    return new Promise(async (resolve, reject) => {
      console.log(`Starting mTLS listener on port ${lport}`);
      if (lport < 1 || 65535 <= lport) {
        reject('Invalid port number');
      }
      try {
        const reqEnvelope = new pb.Envelope();
        reqEnvelope.setType(pb.ClientPB.MsgMtls);
        const mtlsReq = new pb.MTLSReq();
        mtlsReq.setLport(lport);
        reqEnvelope.setData(mtlsReq.serializeBinary());
        const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
        const mtls = pb.MTLS.deserializeBinary(this.decode(resp));
        const job = await this.jobById(mtls.getJobid());
        resolve(job);
      } catch (err) {
        reject(err);
      }
    });
  }

  async startHTTPListener(domain: string, website: string, lport: number): Promise<pb.Job> {
    return new Promise(async (resolve, reject) => {
      try {
        if (lport < 1 || 65535 <= lport) {
          reject('Invalid port number');
        }
        const reqEnvelope = new pb.Envelope();
        reqEnvelope.setType(pb.ClientPB.MsgHttp);
        const httpReq = new pb.HTTPReq();
        httpReq.setLport(lport);
        httpReq.setDomain(domain);
        httpReq.setWebsite(website);
        reqEnvelope.setData(httpReq.serializeBinary());
        const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
        const http = pb.HTTP.deserializeBinary(this.decode(resp));
        const job = await this.jobById(http.getJobid());
        resolve(job);
      } catch (err) {
        reject(err);
      }
    });
  }

  async startHTTPSListener(domain: string, website: string, lport: number, acme: boolean): Promise<pb.Job> {
    return new Promise(async (resolve, reject) => {
      try {
        if (lport < 1 || 65535 <= lport) {
          reject('Invalid port number');
        }
        const reqEnvelope = new pb.Envelope();
        reqEnvelope.setType(pb.ClientPB.MsgHttp);
        const httpReq = new pb.HTTPReq();
        httpReq.setLport(lport);
        httpReq.setDomain(domain);
        httpReq.setWebsite(website);
        httpReq.setSecure(true);
        httpReq.setAcme(acme ? true : false);
        reqEnvelope.setData(httpReq.serializeBinary());
        const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
        const https = pb.HTTP.deserializeBinary(this.decode(resp));
        const job = await this.jobById(https.getJobid());
        resolve(job);
      } catch (err) {
        reject(err);
      }
    });
  }

  async startDNSListener(domains: string[], canaries: boolean): Promise<pb.Job> {
    return new Promise(async (resolve, reject) => {
      try {
        const reqEnvelope = new pb.Envelope();
        reqEnvelope.setType(pb.ClientPB.MsgDns);
        const dnsReq = new pb.DNSReq();
        dnsReq.setDomainsList(domains);
        dnsReq.setCanaries(canaries ? true : false);
        reqEnvelope.setData(dnsReq.serializeBinary());
        const resp: string = await this._ipc.request('rpc_request', this.encode(reqEnvelope));
        const dns = pb.DNS.deserializeBinary(this.decode(resp));
        const job = await this.jobById(dns.getJobid());
        resolve(job);
      } catch (err) {
        reject(err);
      }
    });
  }

}
