import { Routes, RouterModule } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { ActiveConfig } from '../../app-routing-guards.module';

const routes: Routes = [

    // { path: 'infrastructure', component: , canActivate: [ActiveConfig] },

];

export const InfrastructureRoutes: ModuleWithProviders = RouterModule.forChild(routes);
