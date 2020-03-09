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

import 'reflect-metadata';
import '../polyfills';
import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { HttpClientModule } from '@angular/common/http';
import { registerLocaleData } from '@angular/common';
import localeFr from '@angular/common/locales/fr';

import { AppRoutingModule } from './app-routing.module';

import { IPCService } from './providers/ipc.service';
import { ClientService } from './providers/client.service';
import { SliverService } from './providers/sliver.service';
import { JobsService } from './providers/jobs.service';
import { EventsService } from './providers/events.service';

import { BaseMaterialModule } from './base-material';

import { AppComponent } from './app.component';
import { HomeComponent } from './components/home/home.component';
import { SelectServerComponent } from './components/select-server/select-server.component';
import { TopMenuComponent } from './components/top-menu/top-menu.component';
import { SettingsComponent } from './components/settings/settings.component';

import { GenerateModule } from './modules/generate/generate.module';
import { GenerateRoutes } from './modules/generate/generate.routes';

import { SessionsModule } from './modules/sessions/sessions.module';
import { SessionsRoutes } from './modules/sessions/sessions.routes';

import { InfrastructureModule } from './modules/infrastructure/infrastructure.module';
import { InfrastructureRoutes } from './modules/infrastructure/infrastructure.routes';

import { JobsModule } from './modules/jobs/jobs.module';
import { JobsRoutes } from './modules/jobs/jobs.routes';

import { ScriptingModule } from './modules/scripting/scripting.module';
import { ScriptingRoutes } from './modules/scripting/scripting.routes';


@NgModule({
  declarations: [

    // Components
    AppComponent,
    HomeComponent,
    SelectServerComponent,
    TopMenuComponent,
    SettingsComponent,
  ],
  imports: [
    BrowserModule,
    FormsModule,
    ReactiveFormsModule,
    HttpClientModule,
    BrowserAnimationsModule,
    BaseMaterialModule,

    // Routes
    AppRoutingModule,
    GenerateRoutes,
    SessionsRoutes,
    InfrastructureRoutes,
    JobsRoutes,
    ScriptingRoutes,

    // Modules
    GenerateModule,
    SessionsModule,
    InfrastructureModule,
    JobsModule,
    ScriptingModule,

  ],
  providers: [IPCService, ClientService, SliverService, JobsService, EventsService],
  bootstrap: [AppComponent],
  entryComponents: [SelectServerComponent]
})
export class AppModule { }
