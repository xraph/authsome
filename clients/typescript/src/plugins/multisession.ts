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

}

export function multisessionClient(): MultisessionPlugin {
  return new MultisessionPlugin();
}
