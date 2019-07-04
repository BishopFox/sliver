import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { SliverService } from '../../../../providers/sliver.service';
import * as pb from '../../../../../../rpc/pb';


@Component({
  selector: 'app-interact',
  templateUrl: './interact.component.html',
  styleUrls: ['./interact.component.scss']
})
export class InteractComponent implements OnInit {

  session: pb.Sliver;

  constructor(private _route: ActivatedRoute,
              private _sliverSerivce: SliverService) { }

  ngOnInit() {
    this._route.params.subscribe(params => {
      this._sliverSerivce.sessionById(params['id']).then((session) => {
        this.session = session;
      }).catch(() => {
        console.log(`No session with id ${params['id']}`);
      });
    });
  }

}
