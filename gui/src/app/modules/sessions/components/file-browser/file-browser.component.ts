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

import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { MatTableDataSource } from '@angular/material/table';
import { Sort } from '@angular/material/sort';

import { FADE_IN_OUT } from '../../../../shared/animations';
import { SliverService } from '../../../../providers/sliver.service';
import * as pb from '../../../../../../rpc/pb';


interface TableFileData {
  name: string;
  size: number;
  isDir: boolean;
}

function compare(a: number | string | boolean, b: number | string | boolean, isAsc: boolean) {
  return (a < b ? -1 : 1) * (isAsc ? 1 : -1);
}


@Component({
  selector: 'app-file-browser',
  templateUrl: './file-browser.component.html',
  styleUrls: ['./file-browser.component.scss'],
  animations: [FADE_IN_OUT]
})
export class FileBrowserComponent implements OnInit {

  ls: pb.Ls;
  session: pb.Sliver;
  dataSrc = new MatTableDataSource<TableFileData>();
  displayedColumns: string[] = [
    'isDir', 'name', 'size'
  ];
  isFetching = false;

  constructor(private _route: ActivatedRoute,
              private _sliverService: SliverService) { }

  ngOnInit() {
    this._route.parent.params.subscribe((params) => {
      const sessionId: number = parseInt(params['session-id'], 10);
      this._sliverService.sessionById(sessionId).then((session) => {
        this.session = session;
        this.fetchLs('.');
      }).catch(() => {
        console.log(`No session with id ${sessionId}`);
      });
    });
  }

  async fetchLs(targetDir: string) {
    this.isFetching = true;
    this.ls = await this._sliverService.ls(this.session.getId(), targetDir);
    this.dataSrc.data = this.tableData();
    this.isFetching = false;
  }

  async fetchFile(targetFile: string) {

  }

  async onRowSelection(row: TableFileData) {
    if (row.isDir) {
      const pwd = await this._sliverService.cd(this.session.getId(), row.name);
      this.fetchLs(pwd.getPath());
    }
  }

  tableData(): TableFileData[] {
    const dirLs = this.ls.getFilesList();
    const table: TableFileData[] = [];
    table.push({
      name: '..',
      size: 0,
      isDir: true,
    });
    for (let index = 0; index < dirLs.length; index++) {
      table.push({
        name: dirLs[index].getName(),
        size: dirLs[index].getSize(),
        isDir: dirLs[index].getIsdir()
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
        case 'name': return compare(a.name, b.name, isAsc);
        case 'size': return compare(a.size, b.size, isAsc);
        case 'isDir': return compare(a.isDir, b.isDir, isAsc);
        default: return 0;
      }
    });
  }


}
