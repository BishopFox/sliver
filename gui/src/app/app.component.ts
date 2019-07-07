import { Component } from '@angular/core';
import { EventsService, Events } from './providers/events.service';
import { TranslateService } from '@ngx-translate/core';
import { AppConfig } from '../environments/environment';
import { MatSnackBar } from '@angular/material/snack-bar';

import * as pb from '../../rpc/pb';


@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {

  constructor(private _eventsService: EventsService,
              private _snackBar: MatSnackBar,
              private translate: TranslateService) {
    translate.setDefaultLang('en');
    console.log(AppConfig);
    this._eventsService.eventsSubject$.subscribe((event: pb.Event) => {
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
    const snackBarRef = this._snackBar.open(`Session  opened`, 'Interact', {
      duration: 5000,
    });
    snackBarRef.onAction().subscribe(() => {
      console.log('The snack-bar action was triggered!');
    });
  }

}
