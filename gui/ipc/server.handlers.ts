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

IPC-to-RPC converstion methods that talk to the Sliver Server.

*/

import { RPCClient, SliverPB, ClientPB } from '../rpc';
import * as sliverpb from '../rpc/pb/sliver_pb';
import * as clientpb from '../rpc/pb/client_pb';


export class ServerHandlers {

  static server_sessions(rpc: RPCClient, _: string): Promise<Object|null> {
    return new Promise(async (resolve) => {
      const reqEnvelope = new sliverpb.Envelope();
      reqEnvelope.setType(ClientPB.MsgSessions);
      const respEnvelope = await rpc.request(reqEnvelope);
      const sessions = clientpb.Sessions.deserializeBinary(respEnvelope.getData_asU8());
      resolve(sessions.toObject());
    });
  }

  static server_jobs(rpc: RPCClient, _: string): Promise<Object|null> {
    return new Promise(async (resolve) => {
      const reqEnvelope = new sliverpb.Envelope();
      reqEnvelope.setType(ClientPB.MsgJobs);
      const respEnvelope = await rpc.request(reqEnvelope);
      const jobs = clientpb.Jobs.deserializeBinary(respEnvelope.getData_asU8());
      resolve(jobs.toObject());
    });
  }

  static server_generate(rpc: RPCClient, args: string): Promise<Object | null> {
    return new Promise(async (resolve) => {
      const reqEnvelope = new sliverpb.Envelope();
      reqEnvelope.setType(ClientPB.MsgGenerate);
      const generateReq = new clientpb.GenerateReq();
      const sliverConfig = new clientpb.SliverConfig();


      generateReq.setConfig(sliverConfig);
      const data = generateReq.serializeBinary();
      reqEnvelope.setData(data);
      const respEnvelope = await rpc.request(reqEnvelope);
      const generated: clientpb.Generate = clientpb.Generate.deserializeBinary(respEnvelope.getData_asU8());
      resolve(generated.toObject());
    });
  }
}
