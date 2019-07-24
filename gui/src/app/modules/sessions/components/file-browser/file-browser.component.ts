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

import { Component, OnInit, Inject, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { MatTableDataSource } from '@angular/material/table';
import { Sort } from '@angular/material/sort';
import { MatDialog, MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';
import { MatMenuTrigger } from '@angular/material';

import * as pako from 'pako';
import * as pb from '@rpc/pb';
import { FadeInOut } from '@app/shared/animations';
import { SliverService } from '@app/providers/sliver.service';
import { ClientService } from '@app/providers/client.service';



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
  animations: [FadeInOut]
})
export class FileBrowserComponent implements OnInit {

  ls: pb.Ls;
  session: pb.Sliver;
  dataSrc = new MatTableDataSource<TableFileData>();
  displayedColumns: string[] = [
    'isDir', 'name', 'size', 'options'
  ];
  isFetching = false;
  downloading = false;
  showHiddenFiles = true;

  @ViewChild(MatMenuTrigger, { static: false }) contextMenu: MatMenuTrigger;
  contextMenuPosition = { x: '0px', y: '0px' };

  constructor(public dialog: MatDialog,
              private _route: ActivatedRoute,
              private _clientService: ClientService,
              private _sliverService: SliverService) { }

  ngOnInit() {
    this._route.parent.params.subscribe((params) => {
      const sessionId: number = parseInt(params['session-id'], 10);
      this._sliverService.sessionById(sessionId).then((session) => {
        this.session = session;
        this.fetchLs('.');
      }).catch(() => {
        console.error(`No session with id ${sessionId}`);
      });
    });
  }

  async fetchLs(targetDir: string) {
    this.isFetching = true;
    this.ls = await this._sliverService.ls(this.session.getId(), targetDir);
    this.dataSrc.data = this.tableData();
    this.isFetching = false;
  }

  async onRowSelection(row: TableFileData) {
    if (this.isFetching) {
      return;
    }
    if (row.isDir) {
      this.isFetching = true;
      const pwd = await this._sliverService.cd(this.session.getId(), row.name);
      this.fetchLs(pwd.getPath());
    } else {
      const dialogRef = this.dialog.open(DownloadDialogComponent, {
        data: {
          cwd: this.ls.getPath(),
          name: row.name,
          size: row.size,
        }
      });
      dialogRef.afterClosed().subscribe(async (result) => {
        if (result) {
          this.download(result);
        }
      });
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
      const name = dirLs[index].getName();
      if (!this.showHiddenFiles && name.startsWith('.')) {
        continue;
      }
      table.push({
        name: name,
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

  toggleShowHiddenFiles(checked: boolean) {
    this.showHiddenFiles = checked;
    console.log(`Show hidden files: ${this.showHiddenFiles}`);
    this.dataSrc.data = this.tableData();
  }

  rm(event: any, target: TableFileData) {
    event.stopPropagation();
    this.contextMenu.closeMenu();
    const dialogRef = this.dialog.open(RmDialogComponent, {
      data: {
        cwd: this.ls.getPath(),
        name: target.name,
        isDir: target.isDir
      }
    });
    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        console.log(`[rm] ${result.cwd} / ${result.name} (isDir: ${result.isDir})`);
      }
    });
  }

  mkdir() {
    const dialogRef = this.dialog.open(MkdirDialogComponent, {
      data: {
        cwd: this.ls.getPath()
      }
    });
    dialogRef.afterClosed().subscribe(async (result) => {
      if (result) {
        console.log(`[mkdir] ${result.cwd} / ${result.name}`);
      }
    });
  }

  async download(target: TableFileData) {
    this.contextMenu.closeMenu();
    this.downloading = true;
    console.log(`[download] ${target}`);
    const download = await this._sliverService.download(this.session.getId(), target.name);
    let data = download.getData_asU8();
    if (download.getEncoder() === 'gzip') {
      data = pako.ungzip(data);
    }
    this.downloading = false;
    const msg = `Save downloaded file: ${target.name}`;
    const save = await this._clientService.saveFile('Save File', msg, target.name, data);
    console.log(save);
  }

  async upload() {

  }

  async openFile(target: TableFileData) {
    this.contextMenu.closeMenu();
    this.downloading = true;
    const download = await this._sliverService.download(this.session.getId(), target.name);
    let data = download.getData_asU8();
    if (download.getEncoder() === 'gzip') {
      data = pako.ungzip(data);
    }
    this.downloading = false;
  }

  onContextMenu(event: MouseEvent, row: TableFileData) {
    event.preventDefault();
    this.contextMenuPosition.x = event.clientX + 'px';
    this.contextMenuPosition.y = event.clientY + 'px';
    this.contextMenu.menuData = { 'item': row };
    this.contextMenu.openMenu();
  }

}


@Component({
  selector: 'app-mkdir-dialog',
  templateUrl: 'mkdir-dialog.html',
})
export class MkdirDialogComponent implements OnInit {

  result: any;

  constructor(public dialogRef: MatDialogRef<MkdirDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: any) { }

  ngOnInit() {
    this.result = this.data;
  }

  onNoClick(): void {
    this.dialogRef.close();
  }

}


@Component({
  selector: 'app-rm-dialog',
  templateUrl: 'rm-dialog.html',
})
export class RmDialogComponent {

  isConfirmed = false;
  confirmName = '';

  constructor(public dialogRef: MatDialogRef<RmDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: any) { }

  onNoClick(): void {
    this.dialogRef.close();
  }

  checkConfirmed() {
    this.isConfirmed = this.confirmName === this.data.name && this.data.name !== '';
  }

}


@Component({
  selector: 'app-download-dialog',
  templateUrl: 'download-dialog.html',
})
export class DownloadDialogComponent {

  constructor(public dialogRef: MatDialogRef<RmDialogComponent>,
              @Inject(MAT_DIALOG_DATA) public data: any) { }

  onNoClick(): void {
    this.dialogRef.close();
  }

}
