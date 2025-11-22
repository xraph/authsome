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

  async startImpersonation(request: types.StartImpersonation_reqBody): Promise<void> {
    const path = '/start';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async endImpersonation(request: types.EndImpersonation_reqBody): Promise<void> {
    const path = '/end';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getImpersonation(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('GET', path);
  }

  async listImpersonations(): Promise<void> {
    const path = '/';
    return this.client.request<void>('GET', path);
  }

  async listAuditEvents(): Promise<void> {
    const path = '/audit';
    return this.client.request<void>('GET', path);
  }

  async verifyImpersonation(): Promise<void> {
    const path = '/verify';
    return this.client.request<void>('POST', path);
  }

}

export function impersonationClient(): ImpersonationPlugin {
  return new ImpersonationPlugin();
}
