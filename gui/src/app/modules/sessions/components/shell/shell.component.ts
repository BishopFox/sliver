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

import * as pb from '../../../../../../rpc/pb';
import { TunnelService, Tunnel } from '../../../../providers/tunnel.service';
import { Subscription } from 'rxjs';
import * as xterm from 'xterm';


@Component({
  selector: 'app-shell',
  templateUrl: './shell.component.html',
  styleUrls: ['./shell.component.scss']
})
export class ShellComponent implements OnInit {

  @Input() sliver: pb.Sliver;

  constructor(private _tunnelService: TunnelService) { }

  ngOnInit() {

  }

  async openShell() {
    const tun = await this._tunnelService.createTunnel(this.sliver.getId());

  }


}


@Component({
  selector: 'app-terminal',
  templateUrl: '<div #terminal></div>',
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
    this.recvSub = this.tunnel.recv.subscribe((data: Buffer) => {
      console.log(`[terminal] recv: ${data}`);
      this.terminal.writeUtf8(data);
    });

    this.terminal.onData((data: string) => {
      console.log(`[terminal] send: ${data}`);
      this.tunnel.send.next(Buffer.from(data));
    });
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
  }

}

