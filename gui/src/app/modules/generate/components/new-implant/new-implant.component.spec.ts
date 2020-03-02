import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NewImplantComponent } from './new-implant.component';

describe('NewImplantComponent', () => {
  let component: NewImplantComponent;
  let fixture: ComponentFixture<NewImplantComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ NewImplantComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NewImplantComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
