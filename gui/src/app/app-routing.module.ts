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
import { Routes, RouterModule } from '@angular/router';

import { ActiveConfig } from './app-routing-guards.module';
import { HomeComponent } from './components/home/home.component';
import { SelectServerComponent } from './components/select-server/select-server.component';
import { SettingsComponent } from './components/settings/settings.component';


const routes: Routes = [

    { path: '', component: SelectServerComponent },
    { path: 'settings', component: SettingsComponent },

    // Requires active config
    { path: 'home', component: HomeComponent, canActivate: [ActiveConfig] }

];

@NgModule({
    imports: [RouterModule.forRoot(routes, {useHash: true})],
    exports: [RouterModule]
})
export class AppRoutingModule { }
