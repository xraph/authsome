// Auto-generated compliance plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class CompliancePlugin implements ClientPlugin {
  readonly id = 'compliance';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createProfile(request: types.CreateProfileRequest): Promise<types.ErrorResponse> {
    const path = '/profiles';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async createProfileFromTemplate(request: types.CreateProfileFromTemplate_req): Promise<types.ErrorResponse> {
    const path = '/profiles/from-template';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getProfile(): Promise<void> {
    const path = '/profiles/:id';
    return this.client.request<void>('GET', path);
  }

  async getAppProfile(): Promise<void> {
    const path = '/apps/:appId/profile';
    return this.client.request<void>('GET', path);
  }

  async updateProfile(request: types.UpdateProfileRequest): Promise<types.ErrorResponse> {
    const path = '/profiles/:id';
    return this.client.request<types.ErrorResponse>('PUT', path, {
      body: request,
    });
  }

  async deleteProfile(): Promise<void> {
    const path = '/profiles/:id';
    return this.client.request<void>('DELETE', path);
  }

  async getComplianceStatus(): Promise<void> {
    const path = '/apps/:appId/status';
    return this.client.request<void>('GET', path);
  }

  async getDashboard(): Promise<void> {
    const path = '/apps/:appId/dashboard';
    return this.client.request<void>('GET', path);
  }

  async runCheck(request: types.RunCheck_req): Promise<types.ErrorResponse> {
    const path = '/profiles/:profileId/checks';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listChecks(): Promise<void> {
    const path = '/profiles/:profileId/checks';
    return this.client.request<void>('GET', path);
  }

  async getCheck(): Promise<void> {
    const path = '/checks/:id';
    return this.client.request<void>('GET', path);
  }

  async listViolations(): Promise<void> {
    const path = '/apps/:appId/violations';
    return this.client.request<void>('GET', path);
  }

  async getViolation(): Promise<void> {
    const path = '/violations/:id';
    return this.client.request<void>('GET', path);
  }

  async resolveViolation(): Promise<void> {
    const path = '/violations/:id/resolve';
    return this.client.request<void>('PUT', path);
  }

  async generateReport(request: types.GenerateReport_req): Promise<types.ErrorResponse> {
    const path = '/apps/:appId/reports';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listReports(): Promise<void> {
    const path = '/apps/:appId/reports';
    return this.client.request<void>('GET', path);
  }

  async getReport(): Promise<void> {
    const path = '/reports/:id';
    return this.client.request<void>('GET', path);
  }

  async downloadReport(): Promise<types.ErrorResponse> {
    const path = '/reports/:id/download';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async createEvidence(request: types.CreateEvidence_req): Promise<types.ErrorResponse> {
    const path = '/apps/:appId/evidence';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listEvidence(): Promise<void> {
    const path = '/apps/:appId/evidence';
    return this.client.request<void>('GET', path);
  }

  async getEvidence(): Promise<void> {
    const path = '/evidence/:id';
    return this.client.request<void>('GET', path);
  }

  async deleteEvidence(): Promise<void> {
    const path = '/evidence/:id';
    return this.client.request<void>('DELETE', path);
  }

  async createPolicy(request: types.CreatePolicy_req): Promise<types.ErrorResponse> {
    const path = '/apps/:appId/policies';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listPolicies(): Promise<void> {
    const path = '/apps/:appId/policies';
    return this.client.request<void>('GET', path);
  }

  async getPolicy(): Promise<void> {
    const path = '/policies/:id';
    return this.client.request<void>('GET', path);
  }

  async updatePolicy(request: types.UpdatePolicy_req): Promise<types.ErrorResponse> {
    const path = '/policies/:id';
    return this.client.request<types.ErrorResponse>('PUT', path, {
      body: request,
    });
  }

  async deletePolicy(): Promise<void> {
    const path = '/policies/:id';
    return this.client.request<void>('DELETE', path);
  }

  async createTraining(request: types.CreateTraining_req): Promise<types.ErrorResponse> {
    const path = '/apps/:appId/training';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listTraining(): Promise<void> {
    const path = '/apps/:appId/training';
    return this.client.request<void>('GET', path);
  }

  async getUserTraining(): Promise<void> {
    const path = '/users/:userId/training';
    return this.client.request<void>('GET', path);
  }

  async completeTraining(request: types.CompleteTraining_req): Promise<types.ErrorResponse> {
    const path = '/training/:id/complete';
    return this.client.request<types.ErrorResponse>('PUT', path, {
      body: request,
    });
  }

  async listTemplates(): Promise<void> {
    const path = '/templates';
    return this.client.request<void>('GET', path);
  }

  async getTemplate(): Promise<types.ErrorResponse> {
    const path = '/templates/:standard';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

}

export function complianceClient(): CompliancePlugin {
  return new CompliancePlugin();
}
