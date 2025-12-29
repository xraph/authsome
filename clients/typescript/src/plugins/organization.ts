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

  async createOrganization(): Promise<types.Organization> {
    const path = '/organizations/createorganization';
    return this.client.request<types.Organization>('POST', path);
  }

  async updateOrganization(params: { id: string }): Promise<types.Organization> {
    const path = `/organizations/${params.id}`;
    return this.client.request<types.Organization>('PATCH', path);
  }

  async deleteOrganization(params: { id: string }): Promise<void> {
    const path = `/organizations/${params.id}`;
    return this.client.request<void>('DELETE', path);
  }

  async inviteMember(): Promise<void> {
    const path = '/organizations/invite';
    return this.client.request<void>('POST', path);
  }

  async removeMember(params: { memberId: string }): Promise<void> {
    const path = `/organizations/${params.memberId}`;
    return this.client.request<void>('DELETE', path);
  }

  async createTeam(): Promise<void> {
    const path = '/organizations/createteam';
    return this.client.request<void>('POST', path);
  }

  async updateTeam(params: { teamId: string }): Promise<void> {
    const path = `/organizations/${params.teamId}`;
    return this.client.request<void>('PATCH', path);
  }

  async deleteTeam(params: { teamId: string }): Promise<void> {
    const path = `/organizations/${params.teamId}`;
    return this.client.request<void>('DELETE', path);
  }

  async getOrganization(params: { id: string }): Promise<types.Organization> {
    const path = `/organizations/${params.id}`;
    return this.client.request<types.Organization>('GET', path);
  }

  async listOrganizations(): Promise<types.Organization> {
    const path = '/organizations/listorganizations';
    return this.client.request<types.Organization>('GET', path);
  }

  async getOrganizationBySlug(params: { slug: string }): Promise<types.Organization> {
    const path = `/organizations/slug/${params.slug}`;
    return this.client.request<types.Organization>('GET', path);
  }

  async listMembers(): Promise<void> {
    const path = '/organizations/listmembers';
    return this.client.request<void>('GET', path);
  }

  async updateMember(params: { memberId: string }): Promise<void> {
    const path = `/organizations/${params.memberId}`;
    return this.client.request<void>('PATCH', path);
  }

  async acceptInvitation(params: { token: string }): Promise<void> {
    const path = `/organizations/${params.token}/accept`;
    return this.client.request<void>('POST', path);
  }

  async declineInvitation(params: { token: string }): Promise<types.StatusResponse> {
    const path = `/organizations/${params.token}/decline`;
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async listTeams(): Promise<void> {
    const path = '/organizations/listteams';
    return this.client.request<void>('GET', path);
  }

}

export function organizationClient(): OrganizationPlugin {
  return new OrganizationPlugin();
}
