import { NO_ERRORS_SCHEMA } from '@angular/core';
import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { NamespaceDiffModalComponent } from './namespace-diff-modal.component';
import { MetricsService, MockMetricsService } from '../../services/metrics.service';
import { MarkdownService } from 'ngx-markdown';
import { FormBuilder } from '@angular/forms';
import { ApiService } from '../../services/api.service';
import { CatalogService } from '../../../catalog/catalog.service';

class MockApiService {
  constructor() { }
}

class MockCatalogService {
  constructor() { }
}

describe('NamespaceDiffModalComponent', () => {
  let component: NamespaceDiffModalComponent;
  let fixture: ComponentFixture<NamespaceDiffModalComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [NamespaceDiffModalComponent],
      schemas: [NO_ERRORS_SCHEMA],
      providers: [
        { provide: FormBuilder, useClass: FormBuilder },
        { provide: CatalogService, useClass: MockCatalogService },
        { provide: ApiService, useClass: MockApiService },
      ]
    }).compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(NamespaceDiffModalComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });


});
