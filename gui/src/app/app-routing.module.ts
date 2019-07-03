import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

import { ActiveConfig } from './app-routing-guards.module';

import { HomeComponent } from './components/home/home.component';


const routes: Routes = [

    // Requires active config
    { path: '', component: HomeComponent }

];

@NgModule({
    imports: [RouterModule.forRoot(routes, {useHash: true})],
    exports: [RouterModule]
})
export class AppRoutingModule { }
