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


}
