import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { NullablePipe, CapitalizePipe, FileSizePipe } from './pipes';


@NgModule({
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
  ],
  exports: [
    NullablePipe,
    CapitalizePipe,
    FileSizePipe
  ],
  declarations: [
    NullablePipe,
    CapitalizePipe,
    FileSizePipe
  ]
})
export class SharedModule { }
