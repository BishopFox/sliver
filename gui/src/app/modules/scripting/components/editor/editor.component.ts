import { Component, OnInit } from '@angular/core';

@Component({
  selector: 'app-editor',
  templateUrl: './editor.component.html',
  styleUrls: ['./editor.component.scss']
})
export class EditorComponent implements OnInit {

  editorOptions = {
    theme: 'vs',
    fontFamily: 'Source Code Pro',
    fontSize: 13,
    language: 'javascript'
  };
  code = 'function x() {\nconsole.log("Hello world!");\n}';

  constructor() {

  }

  ngOnInit() {

  }

}
