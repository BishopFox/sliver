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

import { SliverService } from '../../../../providers/sliver.service';
import * as pb from '../../../../../../rpc/pb';


@Component({
  selector: 'app-info',
  templateUrl: './info.component.html',
  styleUrls: ['./info.component.scss']
})
export class InfoComponent implements OnInit {

  session: pb.Sliver;
  ifconfig: pb.Ifconfig;

  constructor(private _route: ActivatedRoute,
              private _sliverService: SliverService) { }

  ngOnInit() {
    this._route.parent.params.subscribe((params) => {
      const sessionId: number = parseInt(params['session-id'], 10);
      this._sliverService.sessionById(sessionId).then((session) => {
        this.session = session;
        this.fetchIfconfig();
      }).catch(() => {
        console.error(`No session with id ${sessionId}`);
      });
    });
  }

  async fetchIfconfig() {
    this.ifconfig = await this._sliverService.ifconfig(this.session.getId());
  }

  get interfaces(): pb.NetInterface[] {
    if (!this.ifconfig) {
      return [];
    }
    return this.ifconfig.getNetinterfacesList();
  }

}
