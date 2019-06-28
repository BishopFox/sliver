import { NgModule } from '@angular/core';
import { MatButtonModule, MatCheckboxModule, MatOptionModule, MatSelectModule, MatFormFieldModule } from '@angular/material';

const modules = [MatButtonModule, MatCheckboxModule, MatOptionModule, MatSelectModule, MatFormFieldModule];

@NgModule({
  imports: modules,
  exports: modules,
})
export class BaseMaterialModule { }
