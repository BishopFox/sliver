import { Component } from '@angular/core';
import { SliverService } from './providers/sliver.service';
import { TranslateService } from '@ngx-translate/core';
import { AppConfig } from '../environments/environment';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent {

  constructor(public sliverService: SliverService, private translate: TranslateService) {
    translate.setDefaultLang('en');
  }


}
