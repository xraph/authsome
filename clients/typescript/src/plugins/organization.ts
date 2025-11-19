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

  async getOrganization(): Promise<types.ErrorResponse> {
    const path = '/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async listOrganizations(): Promise<types.ErrorResponse> {
    const path = '/listorganizations';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getOrganizationBySlug(): Promise<types.ErrorResponse> {
    const path = '/slug/:slug';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async listMembers(): Promise<types.ErrorResponse> {
    const path = '/listmembers';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async updateMember(): Promise<types.ErrorResponse> {
    const path = '/:memberId';
    return this.client.request<types.ErrorResponse>('PATCH', path);
  }

  async acceptInvitation(): Promise<types.ErrorResponse> {
    const path = '/:token/accept';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

  async declineInvitation(): Promise<types.StatusResponse> {
    const path = '/:token/decline';
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async listTeams(): Promise<types.ErrorResponse> {
    const path = '/listteams';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

}

export function organizationClient(): OrganizationPlugin {
  return new OrganizationPlugin();
}
