import { Routes, RouterModule } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { ActiveConfig } from '../../app-routing-guards.module';
import { SessionsComponent } from './sessions.component';


const routes: Routes = [

    { path: 'sessions', component: SessionsComponent, canActivate: [ActiveConfig] },

];

export const SessionsRoutes: ModuleWithProviders = RouterModule.forChild(routes);
