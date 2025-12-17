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

  async list(): Promise<types.SessionsResponse> {
    const path = '/list';
    return this.client.request<types.SessionsResponse>('GET', path);
  }

  async setActive(request: types.SetActive_body): Promise<types.SessionTokenResponse> {
    const path = '/set-active';
    return this.client.request<types.SessionTokenResponse>('POST', path, {
      body: request,
    });
  }

  async delete(): Promise<types.StatusResponse> {
    const path = '/delete/{id}';
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async getCurrent(): Promise<types.SessionTokenResponse> {
    const path = '/current';
    return this.client.request<types.SessionTokenResponse>('GET', path);
  }

  async getByID(): Promise<types.SessionTokenResponse> {
    const path = '/{id}';
    return this.client.request<types.SessionTokenResponse>('GET', path);
  }

  async revokeAll(request: types.RevokeAll_body): Promise<void> {
    const path = '/revoke-all';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async revokeOthers(): Promise<void> {
    const path = '/revoke-others';
    return this.client.request<void>('POST', path);
  }

  async refresh(): Promise<types.SessionTokenResponse> {
    const path = '/refresh';
    return this.client.request<types.SessionTokenResponse>('POST', path);
  }

  async getStats(): Promise<void> {
    const path = '/stats';
    return this.client.request<void>('GET', path);
  }

}

export function multisessionClient(): MultisessionPlugin {
  return new MultisessionPlugin();
}
