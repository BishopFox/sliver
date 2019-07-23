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

import { ClientService } from '@app/providers/client.service';
import { RPCConfig } from '@rpc/rpc';


@Component({
  selector: 'app-top-menu',
  templateUrl: './top-menu.component.html',
  styleUrls: ['./top-menu.component.scss']
})
export class TopMenuComponent implements OnInit {

  isConnected = false;
  activeConfig: RPCConfig;

  constructor(private _clientService: ClientService) { }

  ngOnInit() {
    this._clientService.isConnected$.subscribe((state) => {
      this.isConnected = state;
      this.getActiveConfig();
    });
  }

  async getActiveConfig() {
    this.activeConfig = await this._clientService.getActiveConfig();
  }

  onExit() {
    this._clientService.exit();
  }

}
