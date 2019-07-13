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
import { FormGroup, FormBuilder, FormControl, Validators, ValidationErrors } from '@angular/forms';
import { FADE_IN_OUT } from '../../../../shared/animations';
import { SliverService } from '../../../../providers/sliver.service';
import { Subscription } from 'rxjs';
import { JobsService } from '../../../../providers/jobs.service';
import * as pb from '../../../../../../rpc/pb';
import { EventsService } from '../../../../providers/events.service';


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
  animations: [FADE_IN_OUT]
})
export class NewImplantComponent implements OnInit, OnDestroy {

  genTargetForm: FormGroup;
  formSub: Subscription;
  genC2Form: FormGroup;
  compileTimeOptionsForm: FormGroup;

  jobs: pb.Job[];
  jobsSubscription: Subscription;
  listeners: Listener[];

  constructor(private _fb: FormBuilder,
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
      mtls: ['', Validators.compose([
        this.validateMTLSEndpoint
      ])],
      http: ['', Validators.compose([
        this.validateHTTPEndpoint
      ])],
      dns: ['', Validators.compose([
        this.validateDNSEndpoint
      ])],
    }, { validator: this.validateGenC2Form });

    this.compileTimeOptionsForm = this._fb.group({
      reconnect: [60, Validators.compose([
        Validators.required,
      ])],
      maxErrors: [1000, Validators.compose([
        Validators.required,
      ])],
      skipSymbols: [false, Validators.compose([
        Validators.required,
      ])],
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
    return validC2 ? null : {invalidC2: 'You must specify at least one C2 endpoint' };
  }

  get C2s(): C2[] {
    const c2s = [];

    return c2s;
  }

  validateMTLSEndpoint(mtls: FormControl): any {
    return null;
  }

  validateHTTPEndpoint(http: FormControl): any {
    return null;
  }

  validateDNSEndpoint(dns: FormControl): any {
    return null;
  }

  onGenerate() {

  }

}
