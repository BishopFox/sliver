import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { PsComponent } from './ps.component';

describe('PsComponent', () => {
  let component: PsComponent;
  let fixture: ComponentFixture<PsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ PsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(PsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
