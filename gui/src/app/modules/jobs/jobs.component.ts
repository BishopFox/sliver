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

import { FADE_IN_OUT } from '../../shared/animations';
import { JobsService } from '../../providers/jobs.service';
import * as pb from '../../../../rpc/pb';
import { EventsService } from '../../providers/events.service';


interface TableJobData {
  id: number;
  name: string;
  protocol: string;
  port: number;
  description: string;
}

function compare(a: number | string, b: number | string, isAsc: boolean) {
  return (a < b ? -1 : 1) * (isAsc ? 1 : -1);
}


@Component({
  selector: 'app-jobs',
  templateUrl: './jobs.component.html',
  styleUrls: ['./jobs.component.scss'],
  animations: [FADE_IN_OUT]
})
export class JobsComponent implements OnInit, OnDestroy {

  subscription: Subscription;
  dataSrc: MatTableDataSource<TableJobData>;
  displayedColumns: string[] = [
    'id', 'name', 'protocol', 'port', 'description',
  ];

  constructor(private _router: Router,
              private _eventsService: EventsService,
              private _jobsService: JobsService) { }

  ngOnInit() {
    this.fetchJobs();
    this.subscription = this._eventsService.jobs$.subscribe(this.fetchJobs);
  }

  ngOnDestroy() {
    this.subscription.unsubscribe();
  }

  async fetchJobs() {
    const jobs = await this._jobsService.jobs();
    this.dataSrc = new MatTableDataSource(this.tableData(jobs));
  }

  tableData(jobs: pb.Jobs): TableJobData[] {
    const activeJobs = jobs.getActiveList();
    const table: TableJobData[] = [];
    for (let index = 0; index < activeJobs.length; index++) {
      table.push({
        id: activeJobs[index].getId(),
        name: activeJobs[index].getName(),
        protocol: activeJobs[index].getProtocol(),
        port: activeJobs[index].getPort(),
        description: activeJobs[index].getDescription(),
      });
    }
    return table.sort((a, b) => (a.id > b.id) ? 1 : -1);
  }

  applyFilter(filterValue: string) {
    this.dataSrc.filter = filterValue.trim().toLowerCase();
  }

  onRowSelection(row: any) {
    this._router.navigate(['sessions', row.id]);
  }

  // Becauase MatTableDataSource is absolute piece of shit
  sortData(event: Sort) {
    this.dataSrc.data = this.dataSrc.data.slice().sort((a, b) => {
      const isAsc = event.direction === 'asc';
      switch (event.active) {
        case 'id': return compare(a.id, b.id, isAsc);
        case 'name': return compare(a.name, b.name, isAsc);
        case 'protocol': return compare(a.protocol, b.protocol, isAsc);
        case 'port': return compare(a.port, b.protocol, isAsc);
        case 'description': return compare(a.description, b.description, isAsc);
        default: return 0;
      }
    });
  }
}
