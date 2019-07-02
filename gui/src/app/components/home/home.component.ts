import { Component, OnInit } from '@angular/core';
import { ConfigService } from '../../providers/config.service';

export interface Config {
  value: string;
  viewValue: string;
}

@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss']
})
export class HomeComponent implements OnInit {

  private configService: ConfigService;

  constructor(configService: ConfigService) {
    this.configService = configService;
  }

  ngOnInit() {
    console.log('Listing configs ...');
    this.configService.listConfigs().then((configs) => {
      console.log(configs);
    });
  }

}
