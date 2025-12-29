// Auto-generated stepup plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class StepupPlugin implements ClientPlugin {
  readonly id = 'stepup';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async evaluate(request: types.EvaluateRequest): Promise<types.StepUpEvaluationResponse> {
    const path = '/stepup/evaluate';
    return this.client.request<types.StepUpEvaluationResponse>('POST', path, {
      body: request,
    });
  }

  async verify(request: types.VerifyRequest): Promise<types.StepUpVerificationResponse> {
    const path = '/stepup/verify';
    return this.client.request<types.StepUpVerificationResponse>('POST', path, {
      body: request,
    });
  }

  async getRequirement(params: { id: string }): Promise<types.StepUpRequirementResponse> {
    const path = `/stepup/requirements/${params.id}`;
    return this.client.request<types.StepUpRequirementResponse>('GET', path);
  }

  async listPendingRequirements(): Promise<types.StepUpRequirementsResponse> {
    const path = '/stepup/requirements/pending';
    return this.client.request<types.StepUpRequirementsResponse>('GET', path);
  }

  async listVerifications(): Promise<types.StepUpVerificationsResponse> {
    const path = '/stepup/verifications';
    return this.client.request<types.StepUpVerificationsResponse>('GET', path);
  }

  async listRememberedDevices(): Promise<types.StepUpDevicesResponse> {
    const path = '/stepup/devices';
    return this.client.request<types.StepUpDevicesResponse>('GET', path);
  }

  async forgetDevice(params: { id: string }): Promise<types.StepUpStatusResponse> {
    const path = `/stepup/devices/${params.id}`;
    return this.client.request<types.StepUpStatusResponse>('DELETE', path);
  }

  async createPolicy(request: types.StepUpPolicy): Promise<types.StepUpPolicyResponse> {
    const path = '/stepup/policies';
    return this.client.request<types.StepUpPolicyResponse>('POST', path, {
      body: request,
    });
  }

  async listPolicies(): Promise<types.StepUpPoliciesResponse> {
    const path = '/stepup/policies';
    return this.client.request<types.StepUpPoliciesResponse>('GET', path);
  }

  async getPolicy(params: { id: string }): Promise<types.StepUpPolicyResponse> {
    const path = `/stepup/policies/${params.id}`;
    return this.client.request<types.StepUpPolicyResponse>('GET', path);
  }

  async updatePolicy(params: { id: string }, request: types.StepUpPolicy): Promise<types.StepUpPolicyResponse> {
    const path = `/stepup/policies/${params.id}`;
    return this.client.request<types.StepUpPolicyResponse>('PUT', path, {
      body: request,
    });
  }

  async deletePolicy(params: { id: string }): Promise<types.StepUpStatusResponse> {
    const path = `/stepup/policies/${params.id}`;
    return this.client.request<types.StepUpStatusResponse>('DELETE', path);
  }

  async getAuditLogs(): Promise<types.StepUpAuditLogsResponse> {
    const path = '/stepup/audit';
    return this.client.request<types.StepUpAuditLogsResponse>('GET', path);
  }

  async status(): Promise<types.StepUpStatusResponse> {
    const path = '/stepup/status';
    return this.client.request<types.StepUpStatusResponse>('GET', path);
  }

}

export function stepupClient(): StepupPlugin {
  return new StepupPlugin();
}
