import { Routes, RouterModule } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { ActiveConfig } from '../../app-routing-guards.module';
import { NewImplantComponent } from './components/new-implant/new-implant.component';
import { HistoryComponent } from './components/history/history.component';



const routes: Routes = [

    { path: 'generate/new-implant', component: NewImplantComponent, canActivate: [ActiveConfig] },
    { path: 'generate/history', component: HistoryComponent, canActivate: [ActiveConfig] }

];

export const GenerateRoutes: ModuleWithProviders = RouterModule.forChild(routes);
