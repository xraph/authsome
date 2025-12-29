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

  async createConsent(request: types.CreateConsentRequest): Promise<types.ConsentRecordResponse> {
    const path = '/records';
    return this.client.request<types.ConsentRecordResponse>('POST', path, {
      body: request,
    });
  }

  async getConsent(params: { id: string }): Promise<types.ConsentRecordResponse> {
    const path = `/records/${params.id}`;
    return this.client.request<types.ConsentRecordResponse>('GET', path);
  }

  async updateConsent(params: { id: string }, request: types.UpdateConsentRequest): Promise<types.ConsentRecordResponse> {
    const path = `/records/${params.id}`;
    return this.client.request<types.ConsentRecordResponse>('PUT', path, {
      body: request,
    });
  }

  async revokeConsent(params: { id: string }, request: types.UpdateConsentRequest): Promise<types.ConsentStatusResponse> {
    const path = `/revoke/${params.id}`;
    return this.client.request<types.ConsentStatusResponse>('POST', path, {
      body: request,
    });
  }

  async createConsentPolicy(request: types.CreatePolicyRequest): Promise<types.ConsentPolicyResponse> {
    const path = '/policies';
    return this.client.request<types.ConsentPolicyResponse>('POST', path, {
      body: request,
    });
  }

  async getConsentPolicy(params: { id: string }): Promise<types.ConsentPolicyResponse> {
    const path = `/policies/${params.id}`;
    return this.client.request<types.ConsentPolicyResponse>('GET', path);
  }

  async recordCookieConsent(request: types.CookieConsentRequest): Promise<types.ConsentCookieResponse> {
    const path = '/cookies';
    return this.client.request<types.ConsentCookieResponse>('POST', path, {
      body: request,
    });
  }

  async getCookieConsent(): Promise<types.ConsentCookieResponse> {
    const path = '/cookies';
    return this.client.request<types.ConsentCookieResponse>('GET', path);
  }

  async requestDataExport(request: types.DataExportRequestInput): Promise<types.ConsentExportResponse> {
    const path = '/export';
    return this.client.request<types.ConsentExportResponse>('POST', path, {
      body: request,
    });
  }

  async getDataExport(params: { id: string }): Promise<types.ConsentExportResponse> {
    const path = `/export/${params.id}`;
    return this.client.request<types.ConsentExportResponse>('GET', path);
  }

  async downloadDataExport(params: { id: string }): Promise<types.ConsentExportFileResponse> {
    const path = `/export/${params.id}/download`;
    return this.client.request<types.ConsentExportFileResponse>('GET', path);
  }

  async requestDataDeletion(request: types.DataDeletionRequestInput): Promise<types.ConsentDeletionResponse> {
    const path = '/deletion';
    return this.client.request<types.ConsentDeletionResponse>('POST', path, {
      body: request,
    });
  }

  async getDataDeletion(params: { id: string }): Promise<types.ConsentDeletionResponse> {
    const path = `/deletion/${params.id}`;
    return this.client.request<types.ConsentDeletionResponse>('GET', path);
  }

  async approveDeletionRequest(params: { id: string }): Promise<types.ConsentStatusResponse> {
    const path = `/deletion/${params.id}/approve`;
    return this.client.request<types.ConsentStatusResponse>('POST', path);
  }

  async getPrivacySettings(): Promise<types.ConsentSettingsResponse> {
    const path = '/settings';
    return this.client.request<types.ConsentSettingsResponse>('GET', path);
  }

  async updatePrivacySettings(request: types.PrivacySettingsRequest): Promise<types.ConsentSettingsResponse> {
    const path = '/settings';
    return this.client.request<types.ConsentSettingsResponse>('PUT', path, {
      body: request,
    });
  }

  async getConsentAuditLogs(): Promise<types.ConsentAuditLogsResponse> {
    const path = '/audit';
    return this.client.request<types.ConsentAuditLogsResponse>('GET', path);
  }

  async generateConsentReport(): Promise<types.ConsentReportResponse> {
    const path = '/reports';
    return this.client.request<types.ConsentReportResponse>('POST', path);
  }

}

export function consentClient(): ConsentPlugin {
  return new ConsentPlugin();
}
