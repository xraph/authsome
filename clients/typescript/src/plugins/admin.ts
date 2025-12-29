// Auto-generated admin plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class AdminPlugin implements ClientPlugin {
  readonly id = 'admin';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createUser(): Promise<void> {
    const path = '/admin/users';
    return this.client.request<void>('POST', path);
  }

  async listUsers(): Promise<void> {
    const path = '/admin/users';
    return this.client.request<void>('GET', path);
  }

  async deleteUser(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/admin/users/${params.id}`;
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async banUser(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/admin/users/${params.id}/ban`;
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async unbanUser(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/admin/users/${params.id}/unban`;
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async impersonateUser(params: { id: string }): Promise<void> {
    const path = `/admin/users/${params.id}/impersonate`;
    return this.client.request<void>('POST', path);
  }

  async setUserRole(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/admin/users/${params.id}/role`;
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async listSessions(): Promise<void> {
    const path = '/admin/sessions';
    return this.client.request<void>('GET', path);
  }

  async revokeSession(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/admin/sessions/${params.id}`;
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async getStats(): Promise<void> {
    const path = '/admin/stats';
    return this.client.request<void>('GET', path);
  }

  async getAuditLogs(): Promise<void> {
    const path = '/admin/audit-logs';
    return this.client.request<void>('GET', path);
  }

}

export function adminClient(): AdminPlugin {
  return new AdminPlugin();
}
