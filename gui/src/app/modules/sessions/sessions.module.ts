import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { BaseMaterialModule } from '../../base-material';
import { SessionsComponent } from './sessions.component';
import { InteractComponent } from './components/interact/interact.component';

@NgModule({
  declarations: [SessionsComponent, InteractComponent],
  imports: [
    CommonModule,
    BaseMaterialModule
  ]
})
export class SessionsModule { }
