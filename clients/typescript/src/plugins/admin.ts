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

  async createUser(request: types.CreateUser_reqBody): Promise<void> {
    const path = '/users';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async listUsers(): Promise<void> {
    const path = '/users';
    return this.client.request<void>('GET', path);
  }

  async deleteUser(): Promise<types.MessageResponse> {
    const path = '/users/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async banUser(request: types.BanUser_reqBody): Promise<types.MessageResponse> {
    const path = '/users/:id/ban';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async unbanUser(request: types.UnbanUser_reqBody): Promise<types.MessageResponse> {
    const path = '/users/:id/unban';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async impersonateUser(request: types.ImpersonateUser_reqBody): Promise<void> {
    const path = '/users/:id/impersonate';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async setUserRole(request: types.SetUserRole_reqBody): Promise<types.MessageResponse> {
    const path = '/users/:id/role';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async listSessions(): Promise<void> {
    const path = '/sessions';
    return this.client.request<void>('GET', path);
  }

  async revokeSession(): Promise<types.MessageResponse> {
    const path = '/sessions/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async getStats(): Promise<void> {
    const path = '/stats';
    return this.client.request<void>('GET', path);
  }

  async getAuditLogs(): Promise<void> {
    const path = '/audit-logs';
    return this.client.request<void>('GET', path);
  }

}

export function adminClient(): AdminPlugin {
  return new AdminPlugin();
}
