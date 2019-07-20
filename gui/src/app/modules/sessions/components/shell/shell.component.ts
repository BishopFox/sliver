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


import { Component, OnInit, ElementRef, ViewChild, Input, OnDestroy } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { Subscription } from 'rxjs';

import { SliverService } from '../../../../providers/sliver.service';
import * as pb from '../../../../../../rpc/pb';
import { TunnelService, Tunnel } from '../../../../providers/tunnel.service';
import * as xterm from 'xterm';


@Component({
  selector: 'app-shell',
  templateUrl: './shell.component.html',
  styleUrls: ['./shell.component.scss']
})
export class ShellComponent implements OnInit {

  session: pb.Sliver;
  tunnel: Tunnel;
  active = false;

  constructor(private _route: ActivatedRoute,
              private _sliverService: SliverService,
              private _tunnelService: TunnelService) { }

  ngOnInit() {
    this._route.parent.params.subscribe((params) => {
      const sessionId: number = parseInt(params['session-id'], 10);
      this._sliverService.sessionById(sessionId).then((session) => {
        this.session = session;
        this.openShell();
      }).catch(() => {
        console.log(`No session with id ${sessionId}`);
      });
    });
  }

  async openShell() {
    this.tunnel = await this._tunnelService.createTunnel(this.session.getId());
    this.active = true;
  }

}


@Component({
  selector: 'app-terminal',
  template: `<div #terminal></div>`,
  styleUrls: ['./terminal.component.scss']
})
export class TerminalComponent implements OnInit, OnDestroy {

  readonly SCROLLBACK = 100000;

  @Input() tunnel: Tunnel;
  @ViewChild('terminal', { static: false }) el: ElementRef;
  terminal: xterm.Terminal;
  recvSub: Subscription;

  constructor() { }

  ngOnInit() {
    this.createTerminal();
  }

  ngOnDestroy() {
    if (this.recvSub) {
      this.recvSub.unsubscribe();
    }
  }

  createTerminal() {
    this.terminal = new xterm.Terminal({
      cursorBlink: true,
      scrollback: this.SCROLLBACK,
    });
    this.terminal.open(this.el.nativeElement);
    this.recvSub = this.tunnel.recv.subscribe((data: Buffer) => {
      console.log(`[terminal] recv: ${data}`);
      this.terminal.writeUtf8(data);
    });
    this.terminal.onData((data: string) => {
      console.log(`[terminal] send: ${data}`);
      this.tunnel.send.next(Buffer.from(data));
    });
  }

}

