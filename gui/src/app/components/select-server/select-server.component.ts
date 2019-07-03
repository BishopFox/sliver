import { Component, OnInit, Inject } from '@angular/core';

import { MatDialogRef, MAT_DIALOG_DATA } from '@angular/material/dialog';


@Component({
  selector: 'app-select-server',
  templateUrl: './select-server.component.html',
  styleUrls: ['./select-server.component.scss']
})
export class SelectServerComponent implements OnInit {

  constructor(public dialogRef: MatDialogRef<SelectServerComponent>,
              @Inject(MAT_DIALOG_DATA) public data: string) { }

  ngOnInit() { }

}
