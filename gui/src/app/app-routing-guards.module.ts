import { Injectable } from '@angular/core';
import { CanActivate } from '@angular/router';
import { Router } from '@angular/router';

import { ConfigService } from './providers/config.service';


@Injectable({
  providedIn: 'root'
})
export class ActiveConfig implements CanActivate {

  constructor(private _router: Router,
              private _configService: ConfigService) { }

  canActivate(): Promise<boolean> {
    return new Promise(async (resolve) => {
      const activeConfig = await this._configService.getActiveConfig();
      if (activeConfig !== null) {
        resolve(true);
      } else {
        console.log('Client is not authenticated, redirecting to config');
        this._router.navigate(['config']);
        resolve(false);
      }
    });
  }

}
