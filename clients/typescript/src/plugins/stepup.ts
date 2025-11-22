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

  async evaluate(request: types.EvaluateRequest): Promise<void> {
    const path = '/evaluate';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(request: types.VerifyRequest): Promise<void> {
    const path = '/verify';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getRequirement(): Promise<void> {
    const path = '/requirements/:id';
    return this.client.request<void>('GET', path);
  }

  async listPendingRequirements(): Promise<types.RequirementsResponse> {
    const path = '/requirements/pending';
    return this.client.request<types.RequirementsResponse>('GET', path);
  }

  async listVerifications(): Promise<void> {
    const path = '/verifications';
    return this.client.request<void>('GET', path);
  }

  async listRememberedDevices(): Promise<types.StepUpDevicesResponse> {
    const path = '/devices';
    return this.client.request<types.StepUpDevicesResponse>('GET', path);
  }

  async forgetDevice(): Promise<types.ForgetDeviceResponse> {
    const path = '/devices/:id';
    return this.client.request<types.ForgetDeviceResponse>('DELETE', path);
  }

  async createPolicy(request: types.StepUpPolicy): Promise<types.StepUpPolicy> {
    const path = '/policies';
    return this.client.request<types.StepUpPolicy>('POST', path, {
      body: request,
    });
  }

  async listPolicies(): Promise<void> {
    const path = '/policies';
    return this.client.request<void>('GET', path);
  }

  async getPolicy(): Promise<void> {
    const path = '/policies/:id';
    return this.client.request<void>('GET', path);
  }

  async updatePolicy(request: types.StepUpPolicy): Promise<types.StepUpPolicy> {
    const path = '/policies/:id';
    return this.client.request<types.StepUpPolicy>('PUT', path, {
      body: request,
    });
  }

  async deletePolicy(): Promise<void> {
    const path = '/policies/:id';
    return this.client.request<void>('DELETE', path);
  }

  async getAuditLogs(): Promise<void> {
    const path = '/audit';
    return this.client.request<void>('GET', path);
  }

  async status(): Promise<void> {
    const path = '/status';
    return this.client.request<void>('GET', path);
  }

}

export function stepupClient(): StepupPlugin {
  return new StepupPlugin();
}
