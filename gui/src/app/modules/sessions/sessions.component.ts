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

import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';
import { MatTableDataSource } from '@angular/material/table';
import { Sort } from '@angular/material/sort';
import { Subscription } from 'rxjs';

import { FadeInOut } from '../../shared/animations';
import { EventsService } from '../../providers/events.service';
import { SliverService } from '../../providers/sliver.service';
import * as pb from '../../../../rpc/pb';


interface TableSessionData {
  id: number;
  name: string;
  transport: string;
  remoteaddress: string;
  username: string;
  os: string;
  checkin: string;
}

function compare(a: number | string, b: number | string, isAsc: boolean) {
  return (a < b ? -1 : 1) * (isAsc ? 1 : -1);
}


@Component({
  selector: 'app-sessions',
  templateUrl: './sessions.component.html',
  styleUrls: ['./sessions.component.scss'],
  animations: [FadeInOut]
})
export class SessionsComponent implements OnInit, OnDestroy {

  subscription: Subscription;
  dataSrc: MatTableDataSource<TableSessionData>;
  displayedColumns: string[] = [
    'id', 'name', 'transport', 'remoteaddress', 'username', 'os', 'checkin'
  ];

  constructor(private _router: Router,
              private _eventsService: EventsService,
              private _sliverService: SliverService) { }

  ngOnInit() {
    this.fetchSessions();
    this.subscription = this._eventsService.jobs$.subscribe(this.fetchSessions);
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  async fetchSessions() {
    const sessions = await this._sliverService.sessions();
    this.dataSrc = new MatTableDataSource(this.tableData(sessions));
  }

  tableData(sessions: pb.Sessions): TableSessionData[] {
    const slivers = sessions.getSliversList();
    const table: TableSessionData[] = [];
    for (let index = 0; index < slivers.length; index++) {
      table.push({
        id: slivers[index].getId(),
        name: slivers[index].getName(),
        transport: slivers[index].getTransport(),
        remoteaddress: slivers[index].getRemoteaddress(),
        username: slivers[index].getUsername(),
        os: slivers[index].getOs(),
        checkin: slivers[index].getLastcheckin()
      });
    }
    return table.sort((a, b) => (a.id > b.id) ? 1 : -1);
  }

  applyFilter(filterValue: string) {
    this.dataSrc.filter = filterValue.trim().toLowerCase();
  }

  onRowSelection(row: any) {
    this._router.navigate(['sessions', row.id, 'info']);
  }

  // Becauase MatTableDataSource is absolute piece of shit
  sortData(event: Sort) {
    this.dataSrc.data = this.dataSrc.data.slice().sort((a, b) => {
      const isAsc = event.direction === 'asc';
      switch (event.active) {
        case 'id': return compare(a.id, b.id, isAsc);
        case 'name': return compare(a.name, b.name, isAsc);
        case 'transport': return compare(a.transport, b.transport, isAsc);
        case 'remoteaddress': return compare(a.remoteaddress, b.remoteaddress, isAsc);
        case 'username': return compare(a.username, b.username, isAsc);
        case 'os': return compare(a.os, b.os, isAsc);
        case 'checkin': return compare(a.checkin, b.checkin, isAsc);
        default: return 0;
      }
    });
  }

}
