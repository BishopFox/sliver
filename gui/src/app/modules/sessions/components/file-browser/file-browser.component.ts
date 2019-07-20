import { Component, OnInit, Input } from '@angular/core';

import { SliverService } from '../../../../providers/sliver.service';
import * as pb from '../../../../../../rpc/pb';


@Component({
  selector: 'app-file-browser',
  templateUrl: './file-browser.component.html',
  styleUrls: ['./file-browser.component.scss']
})
export class FileBrowserComponent implements OnInit {

  @Input() session: pb.Sliver;

  constructor(private _sliverService: SliverService) { }

  ngOnInit() {

  }

}
