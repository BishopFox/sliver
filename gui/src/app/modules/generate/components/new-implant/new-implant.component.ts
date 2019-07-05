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
import { FADE_IN_OUT } from '../../../../shared/animations';
import { SliverService } from '../../../../providers/sliver.service';

@Component({
  selector: 'app-new-implant',
  templateUrl: './new-implant.component.html',
  styleUrls: ['./new-implant.component.scss'],
  animations: [FADE_IN_OUT]
})
export class NewImplantComponent implements OnInit {

  genTargetForm: FormGroup;
  genC2Form: FormGroup;

  constructor(private _fb: FormBuilder,
              private _sliverService: SliverService) { }

  ngOnInit() {

    this.genTargetForm = this._fb.group({
      os: ['', Validators.compose([
        Validators.required,
      ])],
      arch: ['', Validators.compose([
        Validators.required,
      ])],
      format: ['', Validators.compose([
        Validators.required,
      ])],
    });

    this.genC2Form = this._fb.group({
      mtls: ['', Validators.compose([
        Validators.required,
      ])],
      http: ['', Validators.compose([
        Validators.required,
      ])],
      dns: ['', Validators.compose([
        Validators.required,
      ])],
    });

  }

  onGenerate() {

  }

}
