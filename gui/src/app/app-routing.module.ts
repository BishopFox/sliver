import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

import { ActiveConfig } from './app-routing-guards.module';

import { HomeComponent } from './components/home/home.component';
import { SelectServerComponent } from './components/select-server/select-server.component';


const routes: Routes = [

    { path: '', component: SelectServerComponent },

    // Requires active config
    { path: 'home', component: HomeComponent, canActivate: [ActiveConfig] }

];

@NgModule({
    imports: [RouterModule.forRoot(routes, {useHash: true})],
    exports: [RouterModule]
})
export class AppRoutingModule { }
