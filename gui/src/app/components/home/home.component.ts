import { Component, OnInit } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { FADE_IN_OUT } from '../../shared/animations';

import { ConfigService } from '../../providers/config.service';

export interface Config {
  value: string;
  viewValue: string;
}

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss'],
  animations: [FADE_IN_OUT]
})
export class HomeComponent implements OnInit {

  constructor(private _configService: ConfigService,
              private _fb: FormBuilder) { }

  ngOnInit() {
    console.log('Listing configs ...');
    this._configService.listConfigs().then((configs) => {
      console.log(configs);
    });
  }

}
