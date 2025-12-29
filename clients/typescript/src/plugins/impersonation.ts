// Auto-generated impersonation plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class ImpersonationPlugin implements ClientPlugin {
  readonly id = 'impersonation';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async startImpersonation(): Promise<types.ImpersonationStartResponse> {
    const path = '/impersonation/start';
    return this.client.request<types.ImpersonationStartResponse>('POST', path);
  }

  async endImpersonation(): Promise<types.ImpersonationEndResponse> {
    const path = '/impersonation/end';
    return this.client.request<types.ImpersonationEndResponse>('POST', path);
  }

  async getImpersonation(params: { id: string }): Promise<types.ImpersonationSession> {
    const path = `/impersonation/${params.id}`;
    return this.client.request<types.ImpersonationSession>('GET', path);
  }

  async listImpersonations(): Promise<types.ImpersonationListResponse> {
    const path = '/impersonation/';
    return this.client.request<types.ImpersonationListResponse>('GET', path);
  }

  async listAuditEvents(): Promise<types.ImpersonationAuditResponse> {
    const path = '/impersonation/audit';
    return this.client.request<types.ImpersonationAuditResponse>('GET', path);
  }

  async verifyImpersonation(): Promise<types.ImpersonationVerifyResponse> {
    const path = '/impersonation/verify';
    return this.client.request<types.ImpersonationVerifyResponse>('POST', path);
  }

}

export function impersonationClient(): ImpersonationPlugin {
  return new ImpersonationPlugin();
}
