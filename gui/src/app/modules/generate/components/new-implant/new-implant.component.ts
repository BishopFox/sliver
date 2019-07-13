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
import { FormGroup, FormBuilder, FormControl, Validators } from '@angular/forms';
import { FADE_IN_OUT } from '../../../../shared/animations';
import { SliverService } from '../../../../providers/sliver.service';
import { Subscription } from 'rxjs';

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

  constructor(private _fb: FormBuilder,
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
    }, {validator: this.validateC2});

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

  }

  ngOnDestroy() {
    this.formSub.unsubscribe();
  }

  validateC2(formGroup: FormGroup): any {
    const mtls = formGroup.controls['mtls'].value;
    const http = formGroup.controls['http'].value;
    const dns = formGroup.controls['dns'].value;
    return [mtls, http, dns].some(c2 => c2 !== '') ? null : { invalidC2 : true};
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
