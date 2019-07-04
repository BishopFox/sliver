import { Routes, RouterModule } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { ActiveConfig } from '../../app-routing-guards.module';
import { SessionsComponent } from './sessions.component';
import { InteractComponent } from './components/interact/interact.component';


const routes: Routes = [

    { path: 'sessions', component: SessionsComponent, canActivate: [ActiveConfig] },
    { path: 'sessions/:id', component: InteractComponent, canActivate: [ActiveConfig] },

];

export const SessionsRoutes: ModuleWithProviders = RouterModule.forChild(routes);
