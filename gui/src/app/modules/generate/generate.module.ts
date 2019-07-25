/*
  Sliver Implant Framework
  Copyright (C) 2019  Bishop Fox
  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.
  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.
  You should have received a copy of the GNU General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { BaseMaterialModule } from '../../base-material';
import { NewImplantComponent } from './components/new-implant/new-implant.component';
import { HistoryComponent, RegenerateDialogComponent } from './components/history/history.component';
import { CanariesComponent } from './components/canaries/canaries.component';


@NgModule({
  declarations: [NewImplantComponent, HistoryComponent, RegenerateDialogComponent, CanariesComponent],
  imports: [
    CommonModule,
    FormsModule,
    ReactiveFormsModule,
    BaseMaterialModule
  ],
  entryComponents: [RegenerateDialogComponent]
})
export class GenerateModule { }
