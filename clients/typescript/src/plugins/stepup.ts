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

  async evaluate(request: { currency: string; metadata: any; method: string; resource_type: string; route: string; action: string; amount: number }): Promise<void> {
    const path = '/evaluate';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(request: { device_name: string; ip: string; device_id: string; method: string; remember_device: boolean; requirement_id: string; user_agent: string; challenge_token: string; credential: string }): Promise<void> {
    const path = '/verify';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getRequirement(): Promise<void> {
    const path = '/requirements/:id';
    return this.client.request<void>('GET', path);
  }

  async listPendingRequirements(): Promise<void> {
    const path = '/requirements/pending';
    return this.client.request<void>('GET', path);
  }

  async listVerifications(): Promise<void> {
    const path = '/verifications';
    return this.client.request<void>('GET', path);
  }

  async listRememberedDevices(): Promise<void> {
    const path = '/devices';
    return this.client.request<void>('GET', path);
  }

  async forgetDevice(): Promise<void> {
    const path = '/devices/:id';
    return this.client.request<void>('DELETE', path);
  }

  async createPolicy(request: { description: string; enabled: boolean; metadata: any; name: string; updated_at: string; user_id: string; created_at: string; id: string; org_id: string; priority: number; rules: any }): Promise<{ created_at: string; enabled: boolean; id: string; metadata: any; priority: number; rules: any; user_id: string; description: string; name: string; org_id: string; updated_at: string }> {
    const path = '/policies';
    return this.client.request<{ created_at: string; enabled: boolean; id: string; metadata: any; priority: number; rules: any; user_id: string; description: string; name: string; org_id: string; updated_at: string }>('POST', path, {
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

  async updatePolicy(request: { enabled: boolean; metadata: any; org_id: string; description: string; id: string; name: string; priority: number; rules: any; updated_at: string; user_id: string; created_at: string }): Promise<{ user_id: string; created_at: string; description: string; enabled: boolean; id: string; org_id: string; priority: number; rules: any; updated_at: string; metadata: any; name: string }> {
    const path = '/policies/:id';
    return this.client.request<{ user_id: string; created_at: string; description: string; enabled: boolean; id: string; org_id: string; priority: number; rules: any; updated_at: string; metadata: any; name: string }>('PUT', path, {
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
