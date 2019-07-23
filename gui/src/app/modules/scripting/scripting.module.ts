import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { EditorComponent } from './components/editor/editor.component';
import { FormsModule } from '@angular/forms';
import { BaseMaterialModule } from '../../base-material';

import { MonacoEditorModule } from '@materia-ui/ngx-monaco-editor';


@NgModule({
  declarations: [EditorComponent],
  imports: [
    CommonModule,
    FormsModule,
    BaseMaterialModule,
    MonacoEditorModule,
  ]
})
export class ScriptingModule { }
