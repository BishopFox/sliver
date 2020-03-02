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
import { FormGroup, FormBuilder, Validators } from '@angular/forms';
import { Router } from '@angular/router';

import * as pb from '@rpc/pb';
import { FadeInOut } from '@app/shared/animations';
import { JobsService } from '@app/providers/jobs.service';


@Component({
  selector: 'app-start-listener',
  templateUrl: './start-listener.component.html',
  styleUrls: ['./start-listener.component.scss'],
  animations: [FadeInOut]
})
export class StartListenerComponent implements OnInit {

  selectProtocolForm: FormGroup;
  mtlsOptionsForm: FormGroup;
  httpOptionsForm: FormGroup;
  httpsOptionsForm: FormGroup;
  dnsOptionsForm: FormGroup;

  constructor(private _router: Router,
              private _fb: FormBuilder,
              private _jobsService: JobsService) { }

  ngOnInit() {
    this.selectProtocolForm = this._fb.group({
      protocol: ['mtls', Validators.required]
    });

    this.mtlsOptionsForm = this._fb.group({
      lport: [8888, Validators.required]
    });

    this.httpOptionsForm = this._fb.group({
      lport: [80, Validators.required]
    });

    this.httpsOptionsForm = this._fb.group({
      lport: [443, Validators.required]
    });

    this.dnsOptionsForm = this._fb.group({
      domains: ['', Validators.required],
      canarydomains: ['', Validators.required]
    });
  }

  get protocol(): string {
    return this.selectProtocolForm.controls.protocol.value;
  }

  async startListener() {
    let job: pb.Job;
    let form: any;
    switch (this.protocol) {
      case 'mtls':
        form = this.mtlsOptionsForm.value;
        job = await this._jobsService.startMTLSListener(form.lport);
        break;
      case 'http':
        form = this.httpOptionsForm.value;
        job = await this._jobsService.startHTTPListener(form.domain, form.website, form.lport);
        break;
      case 'https':
        form = this.httpsOptionsForm.value;
        job = await this._jobsService.startHTTPSListener(form.domain, form.website, form.lport, form.acme);
        break;
      case 'dns':
        form = this.dnsOptionsForm.value;
        job = await this._jobsService.startDNSListener(form.domains, form.canarydomains);
        break;
    }

    this._router.navigate(['jobs']);
  }

}
