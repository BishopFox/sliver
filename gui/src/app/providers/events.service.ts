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
import { Subject } from 'rxjs';
import * as pb from '../../../rpc/pb';

export const Events = {
  ServerError: 'server',
  Connected: 'connected',
  Disconnected: 'disconnected',
  Joined: 'joined',
  Left: 'left',
  Canary: 'canary',
  Started: 'started',
  Stopped: 'stopped'
};


@Injectable({
  providedIn: 'root'
})
export class EventsService extends ProtobufService {

  eventsSubject$ = new Subject<pb.Event>();

  players$ = new Subject<pb.Event>();
  jobs$ = new Subject<pb.Event>();
  sessions$ = new Subject<pb.Event>();

  constructor(private _ipc: IPCService) {
    super();
    this._ipc.ipcEventSubject$.subscribe((msg) => {
      try {
        const event = pb.Event.deserializeBinary(this.decode(msg.data));
        this.eventsSubject$.next(event);

        const eventType = event.getEventtype();
        switch (eventType) {

          // Players
          case Events.Joined:
          case Events.Left:
            this.players$.next(event);
            break;

          // Jobs
          case Events.Started:
          case Events.Stopped:
            this.jobs$.next(event);
            break;

          // Sessions
          case Events.Connected:
          case Events.Disconnected:
            this.sessions$.next(event);
            break;

          default:
            console.error(`Unknown event type: '${eventType}'`);
        }

      } catch (err) {
        console.error(err);
      }
    });
  }

}
