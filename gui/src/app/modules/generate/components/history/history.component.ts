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

import { Component, OnInit, Inject } from '@angular/core';
import { MatTableDataSource } from '@angular/material/table';
import { Sort } from '@angular/material/sort';
import { MatDialog, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { FADE_IN_OUT } from '../../../../shared/animations';
import { SliverService } from '../../../../providers/sliver.service';
import * as pb from '../../../../../../rpc/pb';
import { ClientService } from '../../../../providers/client.service';


interface TableSliverBuildData {
  name: string;
  os: string;
  arch: string;
  debug: boolean;
  format: string;
  c2URLs: string[];
}

function compare(a: number | string | boolean, b: number | string | boolean, isAsc: boolean) {
  return (a < b ? -1 : 1) * (isAsc ? 1 : -1);
}


@Component({
  selector: 'app-regenerate-dialog',
  templateUrl: 'regenerate-dialog.html',
})
export class RegenerateDialogComponent {

  constructor(public dialogRef: MatDialogRef<RegenerateDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: any) { }

  onNoClick(): void {
    this.dialogRef.close();
  }

}


@Component({
  selector: 'app-history',
  templateUrl: './history.component.html',
  styleUrls: ['./history.component.scss'],
  animations: [FADE_IN_OUT]
})
export class HistoryComponent implements OnInit {

  dataSrc: MatTableDataSource<TableSliverBuildData>;
  displayedColumns: string[] = [
    'name', 'os', 'arch', 'debug', 'format'
  ];

  constructor(public dialog: MatDialog,
              private _clientService: ClientService,
              private _sliverService: SliverService) { }

  ngOnInit() {
    this.fetchSliverBuilds();
  }

  async fetchSliverBuilds() {
    const sliver_builds = await this._sliverService.sliverBuilds();
    this.dataSrc = new MatTableDataSource(this.tableData(sliver_builds));
  }

  tableData(builds: pb.SliverBuilds): TableSliverBuildData[] {

    // For some reason Google thought it'd be cool to not give you any useful
    // datatypes, and their docs on how to use protobuf 'maps' in JavaScript
    // comes down to "read the code bitch." So we just convert these bullshit
    // types into something useful.

    // .entries() - Returns one of these bullshit unuseful nonsense, but there's an
    // undocumented attribute within this object `.arr_` that contains the actual
    // data we want. It's an array of arrays containing [key, value]'s

    const entries = builds.getConfigsMap().entries().arr_;
    const table: TableSliverBuildData[] = [];
    for (const entry of entries) {
      const name: string = entry[0];
      const config: pb.SliverConfig = entry[1];
      table.push({
        name: name,
        os: config.getGoos(),
        arch: config.getGoarch(),
        debug: config.getDebug(),
        format: this.formatToName(config.getFormat()),
        c2URLs: this.c2sToURLs(config.getC2List())
      });
    }
    return table.sort((a, b) => (a.name > b.name) ? 1 : -1);
  }

  applyFilter(filterValue: string) {
    this.dataSrc.filter = filterValue.trim().toLowerCase();
  }

  onRowSelection(row: any) {
    const dialogRef = this.dialog.open(RegenerateDialogComponent, {
      data: row,
    });
    dialogRef.afterClosed().subscribe(async (targetRow) => {
      console.log(`Regenerate target sliver: ${targetRow.name}`);
      const regen = await this._sliverService.regenerate(targetRow.name);
      if (regen) {
        console.log(regen.toObject());
        const file = regen.getFile();
        const msg = `Save regenerated file ${file.getName()}`;
        const path = await this._clientService.saveFile('Save File', msg, file.getName(), file.getData_asU8());
        console.log(`Saved file to: ${path}`);
      } else {
        console.error(`Failed to regenerate sliver ${targetRow.name}`);
      }
    });
  }

  c2sToURLs(sliverC2s: pb.SliverC2[]): string[] {
    const c2URLs: string[] = [];
    for (let index = 0; index < sliverC2s.length; ++index) {
      c2URLs.push(sliverC2s[index].getUrl());
    }
    return c2URLs;
  }

  formatToName(format: number): string {
    // As defined in `client.proto`
    switch (format) {
      case 0:
        return 'Shared Library';
      case 1:
        return 'Shellcode';
      case 2:
        return 'Executable';
      default:
        return 'Unknown';
    }
  }

  // Becauase MatTableDataSource is absolute piece of shit
  sortData(event: Sort) {
    this.dataSrc.data = this.dataSrc.data.slice().sort((a, b) => {
      const isAsc = event.direction === 'asc';
      switch (event.active) {
        case 'name': return compare(a.name, b.name, isAsc);
        case 'os': return compare(a.os, b.os, isAsc);
        case 'arch': return compare(a.arch, b.arch, isAsc);
        case 'debug': return compare(a.debug, b.debug, isAsc);
        case 'format': return compare(a.format, b.format, isAsc);
        default: return 0;
      }
    });
  }

}
