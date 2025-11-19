// Auto-generated apikey plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class ApikeyPlugin implements ClientPlugin {
  readonly id = 'apikey';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createAPIKey(request: types.CreateAPIKey_reqBody): Promise<types.CreateAPIKeyResponse> {
    const path = '/createapikey';
    return this.client.request<types.CreateAPIKeyResponse>('POST', path, {
      body: request,
    });
  }

  async listAPIKeys(): Promise<void> {
    const path = '/listapikeys';
    return this.client.request<void>('GET', path);
  }

  async getAPIKey(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('GET', path);
  }

  async updateAPIKey(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('PUT', path);
  }

  async deleteAPIKey(): Promise<types.MessageResponse> {
    const path = '/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async rotateAPIKey(): Promise<types.RotateAPIKeyResponse> {
    const path = '/:id/rotate';
    return this.client.request<types.RotateAPIKeyResponse>('POST', path);
  }

  async verifyAPIKey(): Promise<void> {
    const path = '/verify';
    return this.client.request<void>('POST', path);
  }

}

export function apikeyClient(): ApikeyPlugin {
  return new ApikeyPlugin();
}
