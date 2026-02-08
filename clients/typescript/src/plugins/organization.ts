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
    const path = '/organizations';
    return this.client.request<types.Organization>('POST', path);
  }

  async getOrganization(params: { id: string }): Promise<types.Organization> {
    const path = `/organizations/${params.id}`;
    return this.client.request<types.Organization>('GET', path);
  }

  async listOrganizations(): Promise<types.Organization> {
    const path = '/organizations';
    return this.client.request<types.Organization>('GET', path);
  }

  async updateOrganization(params: { id: string }): Promise<types.Organization> {
    const path = `/organizations/${params.id}`;
    return this.client.request<types.Organization>('PATCH', path);
  }

  async deleteOrganization(params: { id: string }): Promise<void> {
    const path = `/organizations/${params.id}`;
    return this.client.request<void>('DELETE', path);
  }

  async getOrganizationBySlug(params: { slug: string }): Promise<types.Organization> {
    const path = `/organizations/slug/${params.slug}`;
    return this.client.request<types.Organization>('GET', path);
  }

  async listMembers(params: { id: string }): Promise<void> {
    const path = `/organizations/${params.id}/members`;
    return this.client.request<void>('GET', path);
  }

  async inviteMember(params: { id: string }): Promise<void> {
    const path = `/organizations/${params.id}/members/invite`;
    return this.client.request<void>('POST', path);
  }

  async updateMember(params: { id: string; memberId: string }): Promise<void> {
    const path = `/organizations/${params.id}/members/${params.memberId}`;
    return this.client.request<void>('PATCH', path);
  }

  async removeMember(params: { id: string; memberId: string }): Promise<void> {
    const path = `/organizations/${params.id}/members/${params.memberId}`;
    return this.client.request<void>('DELETE', path);
  }

  async acceptInvitation(params: { token: string }): Promise<void> {
    const path = `/organization-invitations/${params.token}/accept`;
    return this.client.request<void>('POST', path);
  }

  async declineInvitation(params: { token: string }): Promise<types.StatusResponse> {
    const path = `/organization-invitations/${params.token}/decline`;
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async listTeams(params: { id: string }): Promise<void> {
    const path = `/organizations/${params.id}/teams`;
    return this.client.request<void>('GET', path);
  }

  async createTeam(params: { id: string }): Promise<void> {
    const path = `/organizations/${params.id}/teams`;
    return this.client.request<void>('POST', path);
  }

  async updateTeam(params: { id: string; teamId: string }): Promise<void> {
    const path = `/organizations/${params.id}/teams/${params.teamId}`;
    return this.client.request<void>('PATCH', path);
  }

  async deleteTeam(params: { id: string; teamId: string }): Promise<void> {
    const path = `/organizations/${params.id}/teams/${params.teamId}`;
    return this.client.request<void>('DELETE', path);
  }

}

export function organizationClient(): OrganizationPlugin {
  return new OrganizationPlugin();
}
