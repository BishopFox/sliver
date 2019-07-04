import { Component, OnInit, ViewChild } from '@angular/core';
import { MatSort } from '@angular/material/sort';
import { MatTableDataSource } from '@angular/material/table';

import { FADE_IN_OUT } from '../../shared/animations';
import { SliverService } from '../../providers/sliver.service';
import * as clientpb from '../../../../rpc/pb/client_pb';


@Component({
  selector: 'app-sessions',
  templateUrl: './sessions.component.html',
  styleUrls: ['./sessions.component.scss'],
  animations: [FADE_IN_OUT]
})
export class SessionsComponent implements OnInit {

  @ViewChild(MatSort, {static: true}) sort: MatSort;

  displayedColumns: string[] = [
    'id', 'name', 'transport', 'remote-address', 'username', 'os', 'checkin'
  ];

  sessions: clientpb.Sessions;
  dataSrc = new MatTableDataSource<clientpb.Sliver>([]);

  constructor(private _sliver: SliverService) { }

  ngOnInit() {
    this.dataSrc.sort = this.sort;
    this.getSessions();
  }

  async getSessions() {
    this.sessions = await this._sliver.sessions();
    this.dataSrc.data = this.sessions.getSliversList();
  }

  sortingDataAccessor(data: clientpb.Sliver, sortHeaderId: string) {

  }

  formatUsername(session: clientpb.Sliver): string {
    return `${session.getUsername()}/${session.getHostname()}`;
  }

  formatOperatingSystem(session: clientpb.Sliver) {
    return `${session.getOs()}/${session.getArch()}`;
  }

}
