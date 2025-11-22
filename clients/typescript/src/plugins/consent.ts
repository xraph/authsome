// Auto-generated consent plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class ConsentPlugin implements ClientPlugin {
  readonly id = 'consent';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createConsent(request: types.CreateConsentRequest): Promise<void> {
    const path = '/records';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getConsent(): Promise<void> {
    const path = '/records/:id';
    return this.client.request<void>('GET', path);
  }

  async updateConsent(request: types.UpdateConsentRequest): Promise<void> {
    const path = '/records/:id';
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async revokeConsent(request: types.UpdateConsentRequest): Promise<types.MessageResponse> {
    const path = '/revoke/:id';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async createConsentPolicy(request: types.CreatePolicyRequest): Promise<void> {
    const path = '/policies';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getConsentPolicy(): Promise<void> {
    const path = '/policies/:id';
    return this.client.request<void>('GET', path);
  }

  async recordCookieConsent(request: types.CookieConsentRequest): Promise<void> {
    const path = '/cookies';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getCookieConsent(): Promise<void> {
    const path = '/cookies';
    return this.client.request<void>('GET', path);
  }

  async requestDataExport(request: types.DataExportRequestInput): Promise<void> {
    const path = '/export';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getDataExport(): Promise<void> {
    const path = '/export/:id';
    return this.client.request<void>('GET', path);
  }

  async downloadDataExport(): Promise<void> {
    const path = '/export/:id/download';
    return this.client.request<void>('GET', path);
  }

  async requestDataDeletion(request: types.DataDeletionRequestInput): Promise<void> {
    const path = '/deletion';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getDataDeletion(): Promise<void> {
    const path = '/deletion/:id';
    return this.client.request<void>('GET', path);
  }

  async approveDeletionRequest(): Promise<types.MessageResponse> {
    const path = '/deletion/:id/approve';
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async getPrivacySettings(): Promise<void> {
    const path = '/settings';
    return this.client.request<void>('GET', path);
  }

  async updatePrivacySettings(request: types.PrivacySettingsRequest): Promise<void> {
    const path = '/settings';
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async getConsentAuditLogs(): Promise<void> {
    const path = '/audit';
    return this.client.request<void>('GET', path);
  }

  async generateConsentReport(): Promise<void> {
    const path = '/reports';
    return this.client.request<void>('POST', path);
  }

}

export function consentClient(): ConsentPlugin {
  return new ConsentPlugin();
}
