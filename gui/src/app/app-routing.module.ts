import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

import { ActiveConfig } from './app-routing-guards.module';

import { HomeComponent } from './components/home/home.component';
import { ConfigComponent } from './components/config/config.component';

const routes: Routes = [

    // Can activate without config
    {path: 'config', component: ConfigComponent},

    // Requires active config
    {path: '', component: HomeComponent, canActivate: [ActiveConfig]}

];

@NgModule({
    imports: [RouterModule.forRoot(routes, {useHash: true})],
    exports: [RouterModule]
})
export class AppRoutingModule { }
