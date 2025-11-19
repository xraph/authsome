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

  async createUser(request: types.CreateUser_reqBody): Promise<types.ErrorResponse> {
    const path = '/users';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listUsers(): Promise<types.ErrorResponse> {
    const path = '/users';
    return this.client.request<types.ErrorResponse>('GET', path);
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

  async impersonateUser(request: types.ImpersonateUser_reqBody): Promise<types.ErrorResponse> {
    const path = '/users/:id/impersonate';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async setUserRole(request: types.SetUserRole_reqBody): Promise<types.MessageResponse> {
    const path = '/users/:id/role';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async listSessions(): Promise<types.ErrorResponse> {
    const path = '/sessions';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async revokeSession(): Promise<types.MessageResponse> {
    const path = '/sessions/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async getStats(): Promise<types.ErrorResponse> {
    const path = '/stats';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getAuditLogs(): Promise<types.ErrorResponse> {
    const path = '/audit-logs';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

}

export function adminClient(): AdminPlugin {
  return new AdminPlugin();
}
