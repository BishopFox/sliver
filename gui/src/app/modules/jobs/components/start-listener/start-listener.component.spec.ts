import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { StartListenerComponent } from './start-listener.component';

describe('StartListenerComponent', () => {
  let component: StartListenerComponent;
  let fixture: ComponentFixture<StartListenerComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ StartListenerComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(StartListenerComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
