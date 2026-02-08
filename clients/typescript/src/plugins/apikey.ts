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

  async createAPIKey(request: types.CreateAPIKeyRequest): Promise<types.CreateAPIKeyResponse> {
    const path = '/api-keys';
    return this.client.request<types.CreateAPIKeyResponse>('POST', path, {
      body: request,
    });
  }

  async listAPIKeys(request?: types.ListAPIKeysRequest): Promise<types.ListAPIKeysResponse> {
    const path = '/api-keys';
    return this.client.request<types.ListAPIKeysResponse>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async getAPIKey(params: { id: string }, request?: types.GetAPIKeyRequest): Promise<types.APIKey> {
    const path = `/api-keys/${params.id}`;
    return this.client.request<types.APIKey>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async updateAPIKey(params: { id: string }, request: types.UpdateAPIKeyRequest): Promise<types.APIKey> {
    const path = `/api-keys/${params.id}`;
    return this.client.request<types.APIKey>('PUT', path, {
      body: request,
    });
  }

  async deleteAPIKey(params: { id: string }, request?: types.DeleteAPIKeyRequest): Promise<types.MessageResponse> {
    const path = `/api-keys/${params.id}`;
    return this.client.request<types.MessageResponse>('DELETE', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async rotateAPIKey(params: { id: string }, request: types.RotateAPIKeyRequest): Promise<types.RotateAPIKeyResponse> {
    const path = `/api-keys/${params.id}/rotate`;
    return this.client.request<types.RotateAPIKeyResponse>('POST', path, {
      body: request,
    });
  }

  async verifyAPIKey(request: types.VerifyAPIKeyRequest): Promise<types.VerifyAPIKeyResponse> {
    const path = '/api-keys/verify';
    return this.client.request<types.VerifyAPIKeyResponse>('POST', path, {
      body: request,
    });
  }

}

export function apikeyClient(): ApikeyPlugin {
  return new ApikeyPlugin();
}
