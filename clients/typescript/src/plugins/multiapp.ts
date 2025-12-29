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
    const path = '/apps/createapp';
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
    const path = '/apps/listapps';
    return this.client.request<types.AppsListResponse>('GET', path);
  }

  async removeMember(params: { memberId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.memberId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

  async listMembers(): Promise<types.MembersListResponse> {
    const path = '/apps/listmembers';
    return this.client.request<types.MembersListResponse>('GET', path);
  }

  async inviteMember(request: types.InviteMemberRequest): Promise<types.Invitation> {
    const path = '/apps/invite';
    return this.client.request<types.Invitation>('POST', path, {
      body: request,
    });
  }

  async updateMember(params: { memberId: string }, request: types.UpdateMemberRequest): Promise<types.Member> {
    const path = `/apps/${params.memberId}`;
    return this.client.request<types.Member>('PUT', path, {
      body: request,
    });
  }

  async getInvitation(params: { token: string }): Promise<types.Invitation> {
    const path = `/apps/${params.token}`;
    return this.client.request<types.Invitation>('GET', path);
  }

  async acceptInvitation(params: { token: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.token}/accept`;
    return this.client.request<types.MultitenancyStatusResponse>('POST', path);
  }

  async declineInvitation(params: { token: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.token}/decline`;
    return this.client.request<types.MultitenancyStatusResponse>('POST', path);
  }

  async createTeam(request: types.CreateTeamRequest): Promise<types.Team> {
    const path = '/apps/createteam';
    return this.client.request<types.Team>('POST', path, {
      body: request,
    });
  }

  async getTeam(params: { teamId: string }): Promise<types.Team> {
    const path = `/apps/${params.teamId}`;
    return this.client.request<types.Team>('GET', path);
  }

  async updateTeam(params: { teamId: string }, request: types.UpdateTeamRequest): Promise<types.Team> {
    const path = `/apps/${params.teamId}`;
    return this.client.request<types.Team>('PUT', path, {
      body: request,
    });
  }

  async deleteTeam(params: { teamId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.teamId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

  async listTeams(): Promise<types.TeamsListResponse> {
    const path = '/apps/listteams';
    return this.client.request<types.TeamsListResponse>('GET', path);
  }

  async addTeamMember(params: { teamId: string }, request: types.AddTeamMember_req): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.teamId}/members`;
    return this.client.request<types.MultitenancyStatusResponse>('POST', path, {
      body: request,
    });
  }

  async removeTeamMember(params: { teamId: string; memberId: string }): Promise<types.MultitenancyStatusResponse> {
    const path = `/apps/${params.teamId}/members/${params.memberId}`;
    return this.client.request<types.MultitenancyStatusResponse>('DELETE', path);
  }

}

export function multiappClient(): MultiappPlugin {
  return new MultiappPlugin();
}
