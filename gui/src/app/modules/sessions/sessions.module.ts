import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { BaseMaterialModule } from '../../base-material';
import { SessionsComponent } from './sessions.component';

@NgModule({
  declarations: [SessionsComponent],
  imports: [
    CommonModule,
    BaseMaterialModule
  ]
})
export class SessionsModule { }
