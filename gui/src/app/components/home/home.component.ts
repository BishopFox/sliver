import { Component, OnInit } from '@angular/core';
import { FormBuilder } from '@angular/forms';
import { FADE_IN_OUT } from '../../shared/animations';
import { MatDialog } from '@angular/material/dialog';

import { ConfigService } from '../../providers/config.service';

import { SelectServerComponent } from '../select-server/select-server.component';


@Component({
  selector: 'app-home',
  templateUrl: './home.component.html',
  styleUrls: ['./home.component.scss'],
  animations: [FADE_IN_OUT]
})
export class HomeComponent implements OnInit {

  lhost: string;

  constructor(public dialog: MatDialog,
              private _configService: ConfigService,
              private _fb: FormBuilder) { }

  openDialog(): void {
    const dialogRef = this.dialog.open(SelectServerComponent, {
      width: '250px',
      data: {lhost: this.lhost}
    });

    dialogRef.afterClosed().subscribe(result => {
      if (result === undefined) {
        this.openDialog();
      } else {
        this.lhost = result;
      }
    });
  }

  ngOnInit() {
    this.openDialog();
  }

}
