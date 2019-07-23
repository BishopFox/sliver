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

import { Component, OnInit, OnDestroy } from '@angular/core';
import { FormGroup, FormBuilder, Validators, ValidationErrors } from '@angular/forms';
import { Subscription } from 'rxjs';

import * as pb from '@rpc/pb';
import { FadeInOut } from '@app/shared/animations';
import { SliverService } from '@app/providers/sliver.service';
import { JobsService } from '@app/providers/jobs.service';
import { EventsService } from '@app/providers/events.service';
import { ClientService } from '@app/providers/client.service';


interface Listener {
  job: pb.Job;
  checked: boolean;
}

interface C2 {
  protocol: string;
  domains: string[];
  lport: number;
}


@Component({
  selector: 'app-new-implant',
  templateUrl: './new-implant.component.html',
  styleUrls: ['./new-implant.component.scss'],
  animations: [FadeInOut]
})
export class NewImplantComponent implements OnInit, OnDestroy {

  isGenerating = false;

  genTargetForm: FormGroup;
  formSub: Subscription;
  genC2Form: FormGroup;
  compileTimeOptionsForm: FormGroup;

  jobs: pb.Job[];
  jobsSubscription: Subscription;
  listeners: Listener[];

  constructor(private _fb: FormBuilder,
              private _clientService: ClientService,
              private _eventsService: EventsService,
              private _jobsService: JobsService,
              private _sliverService: SliverService) { }

  ngOnInit() {

    this.genTargetForm = this._fb.group({
      os: ['windows', Validators.compose([
        Validators.required,
      ])],
      arch: ['amd64', Validators.compose([
        Validators.required,
      ])],
      format: ['exe', Validators.compose([
        Validators.required,
      ])],
    });

    this.formSub = this.genTargetForm.controls['os'].valueChanges.subscribe((os) => {
      if (os !== 'windows') {
        this.genTargetForm.controls['format'].setValue('exe');
      }
    });

    this.genC2Form = this._fb.group({
      mtls: [''],
      http: [''],
      dns: [''],
    }, { validator: this.validateGenC2Form });

    this.compileTimeOptionsForm = this._fb.group({
      reconnect: [60],
      maxErrors: [1000],
      skipSymbols: [false],
      debug: [false],
    });

    this.fetchJobs();
    this.jobsSubscription = this._eventsService.jobs$.subscribe(this.fetchJobs);
  }

  ngOnDestroy() {
    if (this.formSub) {
      this.formSub.unsubscribe();
    }
    if (this.jobsSubscription) {
      this.jobsSubscription.unsubscribe();
    }
  }

  async fetchJobs() {
    const jobs = await this._jobsService.jobs();
    const activeJobs = jobs.getActiveList();
    this.listeners = [];
    for (let index = 0; index < activeJobs.length; ++index) {
      if (activeJobs[index].getName() === 'rpc') {
        continue;
      }
      this.listeners.push({
        job: activeJobs[index],
        checked: false,
      });
    }
  }

  validateGenC2Form(formGroup: FormGroup): ValidationErrors {
    const mtls = formGroup.controls['mtls'].value;
    const http = formGroup.controls['http'].value;
    const dns = formGroup.controls['dns'].value;
    const validC2 = [mtls, http, dns].some(c2 => c2 !== '');
    return validC2 ? null : { invalidC2: 'You must specify at least one C2 endpoint' };
  }

  get C2s(): C2[] {
    const c2s = [];

    // Get checked listeners
    this.listeners.forEach((listener) => {
      if (listener.checked) {
        c2s.push({
          protocol: listener.job.getProtocol(),
          lport: listener.job.getPort(),
          domains: []
        });
      }
    });
    c2s.concat(this.mtlsEndpoints);
    return c2s;
  }

  isValidC2Config(): boolean {
    return this.C2s.length ? true : false;
  }

  get mtlsEndpoints(): C2[] {
    const c2s: C2[] = [];
    const mtls = this.genC2Form.controls['mtls'].value;
    const urls = this.parseURLs(mtls);
    urls.forEach((url) => {
      c2s.push({
        protocol: 'mtls',
        lport: url.port ? parseInt(url.port, 10) : 8888,
        domains: [url.hostname]
      });
    });
    return c2s;
  }

  parseURLs(value: string): URL[] {
    const urls: URL[] = [];
    try {
      value.split(',').forEach((rawValue) => {
        if (rawValue === '') {
          return;
        }
        if (rawValue.indexOf('://') !== -1) {
          rawValue = rawValue.slice(rawValue.indexOf('://') + 3, rawValue.length);
        }
        // Basically because JavaScript is a total piece of shit language, if the
        // url is not prefixed with "http" it won't be parsed correctly. Because
        // why would you ever want to parse a non-HTTP URL? Do those even exist?
        const url: URL = new URL(`http://${rawValue}`);
        urls.push(url);
      });
    } catch (err) {
      console.error(err);
    }
    return urls;
  }

  async onGenerate() {
    this.isGenerating = true;
    const config = new pb.SliverConfig();


    const generate = await this._sliverService.generate(config);
    const file = generate.getFile();
    const msg = `Save new implant ${file.getName()}`;
    const save = await this._clientService.saveFile('Save File', msg, file.getName(), file.getData_asU8());
    console.log(`Saved file to: ${save}`);
  }

}
