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

  async startImpersonation(request: types.StartImpersonation_reqBody): Promise<types.ErrorResponse> {
    const path = '/start';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async endImpersonation(request: types.EndImpersonation_reqBody): Promise<types.ErrorResponse> {
    const path = '/end';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getImpersonation(): Promise<types.ErrorResponse> {
    const path = '/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async listImpersonations(): Promise<types.ErrorResponse> {
    const path = '/';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async listAuditEvents(): Promise<types.ErrorResponse> {
    const path = '/audit';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async verifyImpersonation(): Promise<types.ErrorResponse> {
    const path = '/verify';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

}

export function impersonationClient(): ImpersonationPlugin {
  return new ImpersonationPlugin();
}
