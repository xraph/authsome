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

  async createConsent(request: types.CreateConsentRequest): Promise<types.ErrorResponse> {
    const path = '/records';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getConsent(): Promise<types.ErrorResponse> {
    const path = '/records/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async updateConsent(request: types.UpdateConsentRequest): Promise<types.ErrorResponse> {
    const path = '/records/:id';
    return this.client.request<types.ErrorResponse>('PUT', path, {
      body: request,
    });
  }

  async revokeConsent(request: types.UpdateConsentRequest): Promise<types.MessageResponse> {
    const path = '/revoke/:id';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async createConsentPolicy(request: types.CreatePolicyRequest): Promise<types.ErrorResponse> {
    const path = '/policies';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getConsentPolicy(): Promise<types.ErrorResponse> {
    const path = '/policies/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async recordCookieConsent(request: types.CookieConsentRequest): Promise<types.ErrorResponse> {
    const path = '/cookies';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getCookieConsent(): Promise<types.ErrorResponse> {
    const path = '/cookies';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async requestDataExport(request: types.DataExportRequestInput): Promise<types.ErrorResponse> {
    const path = '/export';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getDataExport(): Promise<types.ErrorResponse> {
    const path = '/export/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async downloadDataExport(): Promise<types.ErrorResponse> {
    const path = '/export/:id/download';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async requestDataDeletion(request: types.DataDeletionRequestInput): Promise<types.ErrorResponse> {
    const path = '/deletion';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getDataDeletion(): Promise<types.ErrorResponse> {
    const path = '/deletion/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async approveDeletionRequest(): Promise<types.MessageResponse> {
    const path = '/deletion/:id/approve';
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async getPrivacySettings(): Promise<types.ErrorResponse> {
    const path = '/settings';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async updatePrivacySettings(request: types.PrivacySettingsRequest): Promise<types.ErrorResponse> {
    const path = '/settings';
    return this.client.request<types.ErrorResponse>('PUT', path, {
      body: request,
    });
  }

  async getConsentAuditLogs(): Promise<void> {
    const path = '/audit';
    return this.client.request<void>('GET', path);
  }

  async generateConsentReport(): Promise<types.ErrorResponse> {
    const path = '/reports';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

}

export function consentClient(): ConsentPlugin {
  return new ConsentPlugin();
}
