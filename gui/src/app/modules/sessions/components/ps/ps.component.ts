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

import { Component, OnInit, Input, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { MatTableDataSource } from '@angular/material/table';
import { Sort } from '@angular/material/sort';

import * as pb from '../../../../../../rpc/pb';
import { FADE_IN_OUT } from '../../../../shared/animations';
import { SliverService } from '../../../../providers/sliver.service';


interface TableProcessData {
  pid: number;
  name: string;
  owner: string;
  ppid: number;
}

function compare(a: number | string, b: number | string, isAsc: boolean) {
  return (a < b ? -1 : 1) * (isAsc ? 1 : -1);
}


@Component({
  selector: 'app-ps',
  templateUrl: './ps.component.html',
  styleUrls: ['./ps.component.scss'],
  animations: [FADE_IN_OUT]
})
export class PsComponent implements OnInit, OnDestroy {

  session: pb.Sliver;
  interval: NodeJS.Timer;
  dataSrc: MatTableDataSource<TableProcessData>;
  displayedColumns: string[] = [
    'pid', 'ppid', 'name', 'owner',
  ];

  constructor(private _route: ActivatedRoute,
              private _sliverService: SliverService) { }

  ngOnInit() {
    this._route.parent.params.subscribe((params) => {
      const sessionId: number = parseInt(params['session-id'], 10);
      this._sliverService.sessionById(sessionId).then((session) => {
        this.session = session;
        this.fetchPs();
        this.interval = setInterval(this.fetchPs, 2000);
      }).catch(() => {
        console.log(`No session with id ${sessionId}`);
      });
    });
  }

  ngOnDestroy() {
    if (this.interval) {
      clearInterval(this.interval);
    }
  }

  async fetchPs() {
    if (this.session) {
      const ps = await this._sliverService.ps(this.session.getId());
      this.dataSrc = new MatTableDataSource(this.tableData(ps));
    }
  }

  tableData(ps: pb.Ps): TableProcessData[] {
    const procs = ps.getProcessesList();
    const table: TableProcessData[] = [];
    for (let index = 0; index < procs.length; index++) {
      table.push({
        pid: procs[index].getPid(),
        name: procs[index].getExecutable(),
        owner: procs[index].getOwner(),
        ppid: procs[index].getPpid(),
      });
    }
    return table;
  }

  applyFilter(filterValue: string) {
    this.dataSrc.filter = filterValue.trim().toLowerCase();
  }

  // Becauase MatTableDataSource is absolute piece of shit
  sortData(event: Sort) {
    this.dataSrc.data = this.dataSrc.data.slice().sort((a, b) => {
      const isAsc = event.direction === 'asc';
      switch (event.active) {
        case 'pid': return compare(a.pid, b.pid, isAsc);
        case 'name': return compare(a.name, b.name, isAsc);
        case 'owner': return compare(a.owner, b.owner, isAsc);
        case 'ppid': return compare(a.ppid, b.ppid, isAsc);
        default: return 0;
      }
    });
  }

}
