// Auto-generated multiapp plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class MultiappPlugin implements ClientPlugin {
  readonly id = 'multiapp';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createApp(): Promise<void> {
    const path = '/createapp';
    return this.client.request<void>('POST', path);
  }

  async getApp(): Promise<void> {
    const path = '/:appId';
    return this.client.request<void>('GET', path);
  }

  async updateApp(): Promise<void> {
    const path = '/:appId';
    return this.client.request<void>('PUT', path);
  }

  async deleteApp(): Promise<void> {
    const path = '/:appId';
    return this.client.request<void>('DELETE', path);
  }

  async listApps(): Promise<void> {
    const path = '/listapps';
    return this.client.request<void>('GET', path);
  }

  async removeMember(): Promise<void> {
    const path = '/:memberId';
    return this.client.request<void>('DELETE', path);
  }

  async listMembers(): Promise<void> {
    const path = '/listmembers';
    return this.client.request<void>('GET', path);
  }

  async inviteMember(): Promise<void> {
    const path = '/invite';
    return this.client.request<void>('POST', path);
  }

  async updateMember(): Promise<void> {
    const path = '/:memberId';
    return this.client.request<void>('PUT', path);
  }

  async getInvitation(): Promise<void> {
    const path = '/:token';
    return this.client.request<void>('GET', path);
  }

  async acceptInvitation(): Promise<void> {
    const path = '/:token/accept';
    return this.client.request<void>('POST', path);
  }

  async declineInvitation(): Promise<void> {
    const path = '/:token/decline';
    return this.client.request<void>('POST', path);
  }

  async createTeam(): Promise<void> {
    const path = '/createteam';
    return this.client.request<void>('POST', path);
  }

  async getTeam(): Promise<void> {
    const path = '/:teamId';
    return this.client.request<void>('GET', path);
  }

  async updateTeam(): Promise<void> {
    const path = '/:teamId';
    return this.client.request<void>('PUT', path);
  }

  async deleteTeam(): Promise<void> {
    const path = '/:teamId';
    return this.client.request<void>('DELETE', path);
  }

  async listTeams(): Promise<void> {
    const path = '/listteams';
    return this.client.request<void>('GET', path);
  }

  async addTeamMember(request: types.AddTeamMember_req): Promise<void> {
    const path = '/:teamId/members';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async removeTeamMember(): Promise<void> {
    const path = '/:teamId/members/:memberId';
    return this.client.request<void>('DELETE', path);
  }

}

export function multiappClient(): MultiappPlugin {
  return new MultiappPlugin();
}
