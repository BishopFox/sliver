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

import { Routes, RouterModule } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { ActiveConfig } from '../../app-routing-guards.module';
import { SessionsComponent } from './sessions.component';
import { InteractComponent } from './components/interact/interact.component';
import { InfoComponent } from './components/info/info.component';
import { PsComponent } from './components/ps/ps.component';
import { FileBrowserComponent } from './components/file-browser/file-browser.component';
import { ShellComponent } from './components/shell/shell.component';


const routes: Routes = [

    { path: 'sessions', component: SessionsComponent, canActivate: [ActiveConfig] },
    { path: 'sessions/:session-id', component: InteractComponent, canActivate: [ActiveConfig],
      children: [
        { path: 'info', component: InfoComponent, canActivate: [ActiveConfig] },
        { path: 'ps', component: PsComponent, canActivate: [ActiveConfig] },
        { path: 'file-browser', component: FileBrowserComponent, canActivate: [ActiveConfig] },
        { path: 'shell', component: ShellComponent, canActivate: [ActiveConfig] },
      ]
  },

];

export const SessionsRoutes: ModuleWithProviders = RouterModule.forChild(routes);
