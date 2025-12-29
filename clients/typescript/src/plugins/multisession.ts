// Auto-generated multisession plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class MultisessionPlugin implements ClientPlugin {
  readonly id = 'multisession';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async list(request?: types.ListSessionsRequest): Promise<types.ListSessionsResponse> {
    const path = '/multi-session/list';
    return this.client.request<types.ListSessionsResponse>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async setActive(request: types.SetActiveRequest): Promise<types.SessionTokenResponse> {
    const path = '/multi-session/set-active';
    return this.client.request<types.SessionTokenResponse>('POST', path, {
      body: request,
    });
  }

  async delete(params: { id: string }): Promise<types.StatusResponse> {
    const path = `/multi-session/delete/${params.id}`;
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async getCurrent(): Promise<types.SessionTokenResponse> {
    const path = '/multi-session/current';
    return this.client.request<types.SessionTokenResponse>('GET', path);
  }

  async getByID(params: { id: string }): Promise<types.SessionTokenResponse> {
    const path = `/multi-session/${params.id}`;
    return this.client.request<types.SessionTokenResponse>('GET', path);
  }

  async revokeAll(request: types.RevokeAllRequest): Promise<types.RevokeResponse> {
    const path = '/multi-session/revoke-all';
    return this.client.request<types.RevokeResponse>('POST', path, {
      body: request,
    });
  }

  async revokeOthers(): Promise<types.RevokeResponse> {
    const path = '/multi-session/revoke-others';
    return this.client.request<types.RevokeResponse>('POST', path);
  }

  async refresh(): Promise<types.SessionTokenResponse> {
    const path = '/multi-session/refresh';
    return this.client.request<types.SessionTokenResponse>('POST', path);
  }

  async getStats(): Promise<types.SessionStatsResponse> {
    const path = '/multi-session/stats';
    return this.client.request<types.SessionStatsResponse>('GET', path);
  }

}

export function multisessionClient(): MultisessionPlugin {
  return new MultisessionPlugin();
}
