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

import { Component } from '@angular/core';
import { EventsService, Events } from './providers/events.service';
import { MatSnackBar } from '@angular/material/snack-bar';
import { Router } from '@angular/router';

import { AppConfig } from '../environments/environment';
import * as pb from '@rpc/pb';


@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {

  constructor(private _router: Router,
              private _eventsService: EventsService,
              private _snackBar: MatSnackBar) {
    console.log(AppConfig);
    this._eventsService.events$.subscribe((event: pb.Event) => {
      const eventType = event.getEventtype();
      switch (eventType) {

        // Players
        case Events.Joined:
          this.playerAlert('joined', event.getClient());
          break;
        case Events.Left:
          this.playerAlert('left', event.getClient());
          break;

        // Jobs
        case Events.Stopped:
          this.jobStoppedAlert(event.getJob());
          break;

        // Sessions
        case Events.Connected:
          this.sessionOpenedAlert(event.getSliver());
          break;

        default:
          console.error(`Unknown event type: '${eventType}'`);
      }
    });
  }

  playerAlert(action: string, client: pb.Client) {
    this._snackBar.open(`${client.getOperator()} has ${action} the game!`, 'Dismiss', {
      duration: 5000,
    });
  }

  jobStoppedAlert(job: pb.Job) {
    this._snackBar.open(`Job #${job.getId()} (${job.getProtocol()}/${job.getName()}) has stopped.`, 'Dismiss', {
      duration: 5000,
    });
  }

  sessionOpenedAlert(session: pb.Sliver) {
    const snackBarRef = this._snackBar.open(`Session #${session.getId()} opened`, 'Interact', {
      duration: 5000,
    });
    snackBarRef.onAction().subscribe(() => {
      this._router.navigate(['sessions', session.getId()]);
    });

    const _ = new Notification('Sliver', {
      body: `Session #${session.getId()} opened`
    });
  }

}
