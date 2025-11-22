// Auto-generated organization plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class OrganizationPlugin implements ClientPlugin {
  readonly id = 'organization';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createOrganization(): Promise<void> {
    const path = '/createorganization';
    return this.client.request<void>('POST', path);
  }

  async updateOrganization(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('PATCH', path);
  }

  async deleteOrganization(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('DELETE', path);
  }

  async inviteMember(): Promise<void> {
    const path = '/invite';
    return this.client.request<void>('POST', path);
  }

  async removeMember(): Promise<void> {
    const path = '/:memberId';
    return this.client.request<void>('DELETE', path);
  }

  async createTeam(): Promise<void> {
    const path = '/createteam';
    return this.client.request<void>('POST', path);
  }

  async updateTeam(): Promise<void> {
    const path = '/:teamId';
    return this.client.request<void>('PATCH', path);
  }

  async deleteTeam(): Promise<void> {
    const path = '/:teamId';
    return this.client.request<void>('DELETE', path);
  }

  async getOrganization(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('GET', path);
  }

  async listOrganizations(): Promise<void> {
    const path = '/listorganizations';
    return this.client.request<void>('GET', path);
  }

  async getOrganizationBySlug(): Promise<void> {
    const path = '/slug/:slug';
    return this.client.request<void>('GET', path);
  }

  async listMembers(): Promise<void> {
    const path = '/listmembers';
    return this.client.request<void>('GET', path);
  }

  async updateMember(): Promise<void> {
    const path = '/:memberId';
    return this.client.request<void>('PATCH', path);
  }

  async acceptInvitation(): Promise<void> {
    const path = '/:token/accept';
    return this.client.request<void>('POST', path);
  }

  async declineInvitation(): Promise<types.StatusResponse> {
    const path = '/:token/decline';
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async listTeams(): Promise<void> {
    const path = '/listteams';
    return this.client.request<void>('GET', path);
  }

}

export function organizationClient(): OrganizationPlugin {
  return new OrganizationPlugin();
}
