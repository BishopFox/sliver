import { Routes, RouterModule } from '@angular/router';
import { ModuleWithProviders } from '@angular/core';

import { ActiveConfig } from '../../app-routing-guards.module';
import { JobsComponent } from './jobs.component';


const routes: Routes = [

    { path: 'jobs', component: JobsComponent, canActivate: [ActiveConfig] },

];

export const JobsRoutes: ModuleWithProviders = RouterModule.forChild(routes);
