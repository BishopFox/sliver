import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { NewImplantComponent } from './components/new-implant/new-implant.component';
import { HistoryComponent } from './components/history/history.component';

@NgModule({
  declarations: [NewImplantComponent, HistoryComponent],
  imports: [
    CommonModule
  ]
})
export class GenerateModule { }
