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

  async createApp(): Promise<types.App> {
    const path = '/apps';
    return this.client.request<types.App>('POST', path);
  }

  async getApp(params: { appId: string }): Promise<types.App> {
    const path = `/apps/${params.appId}`;
    return this.client.request<types.App>('GET', path);
  }

  async updateApp(params: { appId: string }): Promise<types.App> {
    const path = `/apps/${params.appId}`;
    return this.client.request<types.App>('PUT', path);
  }

  async deleteApp(params: { appId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.appId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

  async listApps(): Promise<types.AppsListResponse> {
    const path = '/apps';
    return this.client.request<types.AppsListResponse>('GET', path);
  }

  async removeMember(params: { appId: string; memberId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.appId}/members/${params.memberId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

  async listMembers(params: { appId: string }): Promise<types.MembersListResponse> {
    const path = `/apps/${params.appId}/members`;
    return this.client.request<types.MembersListResponse>('GET', path);
  }

  async inviteMember(params: { appId: string }, request: types.InviteMemberRequest): Promise<types.Invitation> {
    const path = `/apps/${params.appId}/members/invite`;
    return this.client.request<types.Invitation>('POST', path, {
      body: request,
    });
  }

  async updateMember(params: { appId: string; memberId: string }, request: types.UpdateMemberRequest): Promise<types.Member> {
    const path = `/apps/${params.appId}/members/${params.memberId}`;
    return this.client.request<types.Member>('PUT', path, {
      body: request,
    });
  }

  async getInvitation(params: { token: string }): Promise<types.Invitation> {
    const path = `/invitations/${params.token}`;
    return this.client.request<types.Invitation>('GET', path);
  }

  async acceptInvitation(params: { token: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/invitations/${params.token}/accept`;
    return this.client.request<types.MultitenancyStatusResponse>('POST', path);
  }

  async declineInvitation(params: { token: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/invitations/${params.token}/decline`;
    return this.client.request<types.MultitenancyStatusResponse>('POST', path);
  }

  async createTeam(params: { appId: string }, request: types.CreateTeamRequest): Promise<types.Team> {
    const path = `/apps/${params.appId}/teams`;
    return this.client.request<types.Team>('POST', path, {
      body: request,
    });
  }

  async getTeam(params: { appId: string; teamId: string }): Promise<types.Team> {
    const path = `/apps/${params.appId}/teams/${params.teamId}`;
    return this.client.request<types.Team>('GET', path);
  }

  async updateTeam(params: { appId: string; teamId: string }, request: types.UpdateTeamRequest): Promise<types.Team> {
    const path = `/apps/${params.appId}/teams/${params.teamId}`;
    return this.client.request<types.Team>('PUT', path, {
      body: request,
    });
  }

  async deleteTeam(params: { appId: string; teamId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.appId}/teams/${params.teamId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

  async listTeams(params: { appId: string }): Promise<types.TeamsListResponse> {
    const path = `/apps/${params.appId}/teams`;
    return this.client.request<types.TeamsListResponse>('GET', path);
  }

  async addTeamMember(params: { teamId: string; appId: string }, request: types.AddTeamMember_req): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.appId}/teams/${params.teamId}/members`;
    return this.client.request<types.MultitenancyStatusResponse>('POST', path, {
      body: request,
    });
  }

  async removeTeamMember(params: { appId: string; teamId: string; memberId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.appId}/teams/${params.teamId}/members/${params.memberId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

}

export function multiappClient(): MultiappPlugin {
  return new MultiappPlugin();
}
