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

  async createProfile(request: types.CreateProfileRequest): Promise<types.ComplianceProfileResponse> {
    const path = '/profiles';
    return this.client.request<types.ComplianceProfileResponse>('POST', path, {
      body: request,
    });
  }

  async createProfileFromTemplate(request: types.CreateProfileFromTemplateRequest): Promise<types.ComplianceProfileResponse> {
    const path = '/profiles/from-template';
    return this.client.request<types.ComplianceProfileResponse>('POST', path, {
      body: request,
    });
  }

  async getProfile(params: { id: string }): Promise<types.ComplianceProfileResponse> {
    const path = `/profiles/${params.id}`;
    return this.client.request<types.ComplianceProfileResponse>('GET', path);
  }

  async getAppProfile(params: { appId: string }): Promise<types.ComplianceProfileResponse> {
    const path = `/apps/${params.appId}/profile`;
    return this.client.request<types.ComplianceProfileResponse>('GET', path);
  }

  async updateProfile(params: { id: string }, request: types.UpdateProfileRequest): Promise<types.ComplianceProfileResponse> {
    const path = `/profiles/${params.id}`;
    return this.client.request<types.ComplianceProfileResponse>('PUT', path, {
      body: request,
    });
  }

  async deleteProfile(params: { id: string }): Promise<types.ComplianceStatusResponse> {
    const path = `/profiles/${params.id}`;
    return this.client.request<types.ComplianceStatusResponse>('DELETE', path);
  }

  async getComplianceStatus(params: { appId: string }): Promise<types.ComplianceStatusDetailsResponse> {
    const path = `/apps/${params.appId}/status`;
    return this.client.request<types.ComplianceStatusDetailsResponse>('GET', path);
  }

  async getDashboard(params: { appId: string }): Promise<types.ComplianceDashboardResponse> {
    const path = `/apps/${params.appId}/dashboard`;
    return this.client.request<types.ComplianceDashboardResponse>('GET', path);
  }

  async runCheck(params: { profileId: string }, request: types.RunCheckRequest): Promise<types.ComplianceCheckResponse> {
    const path = `/profiles/${params.profileId}/checks`;
    return this.client.request<types.ComplianceCheckResponse>('POST', path, {
      body: request,
    });
  }

  async listChecks(params: { profileId: string }): Promise<types.ComplianceChecksResponse> {
    const path = `/profiles/${params.profileId}/checks`;
    return this.client.request<types.ComplianceChecksResponse>('GET', path);
  }

  async getCheck(params: { id: string }): Promise<types.ComplianceCheckResponse> {
    const path = `/checks/${params.id}`;
    return this.client.request<types.ComplianceCheckResponse>('GET', path);
  }

  async listViolations(params: { appId: string }): Promise<types.ComplianceViolationsResponse> {
    const path = `/apps/${params.appId}/violations`;
    return this.client.request<types.ComplianceViolationsResponse>('GET', path);
  }

  async getViolation(params: { id: string }): Promise<types.ComplianceViolationResponse> {
    const path = `/violations/${params.id}`;
    return this.client.request<types.ComplianceViolationResponse>('GET', path);
  }

  async resolveViolation(params: { id: string }, request: types.ResolveViolationRequest): Promise<types.ComplianceStatusResponse> {
    const path = `/violations/${params.id}/resolve`;
    return this.client.request<types.ComplianceStatusResponse>('PUT', path, {
      body: request,
    });
  }

  async generateReport(params: { appId: string }, request: types.GenerateReportRequest): Promise<types.ComplianceReportResponse> {
    const path = `/apps/${params.appId}/reports`;
    return this.client.request<types.ComplianceReportResponse>('POST', path, {
      body: request,
    });
  }

  async listReports(params: { appId: string }): Promise<types.ComplianceReportsResponse> {
    const path = `/apps/${params.appId}/reports`;
    return this.client.request<types.ComplianceReportsResponse>('GET', path);
  }

  async getReport(params: { id: string }): Promise<types.ComplianceReportResponse> {
    const path = `/reports/${params.id}`;
    return this.client.request<types.ComplianceReportResponse>('GET', path);
  }

  async downloadReport(params: { id: string }): Promise<types.ComplianceReportFileResponse> {
    const path = `/reports/${params.id}/download`;
    return this.client.request<types.ComplianceReportFileResponse>('GET', path);
  }

  async createEvidence(params: { appId: string }, request: types.CreateEvidenceRequest): Promise<types.ComplianceEvidenceResponse> {
    const path = `/apps/${params.appId}/evidence`;
    return this.client.request<types.ComplianceEvidenceResponse>('POST', path, {
      body: request,
    });
  }

  async listEvidence(params: { appId: string }): Promise<types.ComplianceEvidencesResponse> {
    const path = `/apps/${params.appId}/evidence`;
    return this.client.request<types.ComplianceEvidencesResponse>('GET', path);
  }

  async getEvidence(params: { id: string }): Promise<types.ComplianceEvidenceResponse> {
    const path = `/evidence/${params.id}`;
    return this.client.request<types.ComplianceEvidenceResponse>('GET', path);
  }

  async deleteEvidence(params: { id: string }): Promise<types.ComplianceStatusResponse> {
    const path = `/evidence/${params.id}`;
    return this.client.request<types.ComplianceStatusResponse>('DELETE', path);
  }

  async createPolicy(params: { appId: string }, request: types.CreatePolicyRequest): Promise<types.CompliancePolicyResponse> {
    const path = `/apps/${params.appId}/policies`;
    return this.client.request<types.CompliancePolicyResponse>('POST', path, {
      body: request,
    });
  }

  async listPolicies(params: { appId: string }): Promise<types.CompliancePoliciesResponse> {
    const path = `/apps/${params.appId}/policies`;
    return this.client.request<types.CompliancePoliciesResponse>('GET', path);
  }

  async getPolicy(params: { id: string }): Promise<types.CompliancePolicyResponse> {
    const path = `/policies/${params.id}`;
    return this.client.request<types.CompliancePolicyResponse>('GET', path);
  }

  async updatePolicy(params: { id: string }, request: types.UpdatePolicyRequest): Promise<types.CompliancePolicyResponse> {
    const path = `/policies/${params.id}`;
    return this.client.request<types.CompliancePolicyResponse>('PUT', path, {
      body: request,
    });
  }

  async deletePolicy(params: { id: string }): Promise<types.ComplianceStatusResponse> {
    const path = `/policies/${params.id}`;
    return this.client.request<types.ComplianceStatusResponse>('DELETE', path);
  }

  async createTraining(params: { appId: string }, request: types.CreateTrainingRequest): Promise<types.ComplianceTrainingResponse> {
    const path = `/apps/${params.appId}/training`;
    return this.client.request<types.ComplianceTrainingResponse>('POST', path, {
      body: request,
    });
  }

  async listTraining(params: { appId: string }): Promise<types.ComplianceTrainingsResponse> {
    const path = `/apps/${params.appId}/training`;
    return this.client.request<types.ComplianceTrainingsResponse>('GET', path);
  }

  async getUserTraining(params: { userId: string }): Promise<types.ComplianceUserTrainingResponse> {
    const path = `/users/${params.userId}/training`;
    return this.client.request<types.ComplianceUserTrainingResponse>('GET', path);
  }

  async completeTraining(params: { id: string }, request: types.CompleteTrainingRequest): Promise<types.ComplianceStatusResponse> {
    const path = `/training/${params.id}/complete`;
    return this.client.request<types.ComplianceStatusResponse>('PUT', path, {
      body: request,
    });
  }

  async listTemplates(): Promise<types.ComplianceTemplatesResponse> {
    const path = '/templates';
    return this.client.request<types.ComplianceTemplatesResponse>('GET', path);
  }

  async getTemplate(params: { standard: string }): Promise<types.ComplianceTemplateResponse> {
    const path = `/templates/${params.standard}`;
    return this.client.request<types.ComplianceTemplateResponse>('GET', path);
  }

}

export function complianceClient(): CompliancePlugin {
  return new CompliancePlugin();
}
