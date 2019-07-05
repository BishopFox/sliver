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
------------------------------------------------------------------------

Easy access to methods used to en/decode (base64) IPC protobuf messages.

*/

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
