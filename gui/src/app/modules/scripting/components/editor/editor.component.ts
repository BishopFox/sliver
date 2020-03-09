import { Component, OnInit } from '@angular/core';
import { IPCService } from '@app/providers/ipc.service';

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

  constructor(private _ipcService: IPCService) { }

  ngOnInit() {

  }

  async executeScript() {
    const scriptReq = JSON.stringify({
      script: this.code,
      devtools: true,
    });
    const scriptId = await this._ipcService.request('client_executeScript', scriptReq);
    console.log(`Executed script with id ${scriptId}`);
  }

}
