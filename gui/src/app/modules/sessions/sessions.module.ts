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

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { BaseMaterialModule } from '../../base-material';
import { RouterModule } from '@angular/router';
import { SessionsComponent } from './sessions.component';
import { InteractComponent } from './components/interact/interact.component';
import { FileBrowserComponent } from './components/file-browser/file-browser.component';
import { ShellComponent, TerminalComponent } from './components/shell/shell.component';
import { PsComponent } from './components/ps/ps.component';
import { InfoComponent } from './components/info/info.component';

@NgModule({
  declarations: [
    SessionsComponent,
    InteractComponent,
    FileBrowserComponent,
    ShellComponent,
    TerminalComponent,
    PsComponent,
    InfoComponent
  ],
  imports: [
    CommonModule,
    RouterModule,
    BaseMaterialModule
  ]
})
export class SessionsModule { }
