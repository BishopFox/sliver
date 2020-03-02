import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { CanariesComponent } from './canaries.component';

describe('CanariesComponent', () => {
  let component: CanariesComponent;
  let fixture: ComponentFixture<CanariesComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ CanariesComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(CanariesComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
