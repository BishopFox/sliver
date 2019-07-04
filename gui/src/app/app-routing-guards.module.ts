import { Injectable } from '@angular/core';
import { CanActivate } from '@angular/router';
import { Router } from '@angular/router';

import { ClientService } from './providers/client.service';


@Injectable({
  providedIn: 'root'
})
export class ActiveConfig implements CanActivate {

  constructor(private _router: Router,
              private _clientService: ClientService) { }

  canActivate(): Promise<boolean> {
    return new Promise(async (resolve) => {
      const activeConfig = await this._clientService.getActiveConfig();
      if (activeConfig !== null) {
        resolve(true);
      } else {
        this._router.navigate(['']);
        resolve(false);
      }
    });
  }

}
