// Auto-generated secrets plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class SecretsPlugin implements ClientPlugin {
  readonly id = 'secrets';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async list(): Promise<types.ListSecretsResponse> {
    const path = '/list';
    return this.client.request<types.ListSecretsResponse>('GET', path);
  }

  async create(): Promise<types.ErrorResponse> {
    const path = '/create';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

  async get(): Promise<types.ErrorResponse> {
    const path = '/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getValue(): Promise<types.RevealValueResponse> {
    const path = '/:id/value';
    return this.client.request<types.RevealValueResponse>('GET', path);
  }

  async update(): Promise<types.ErrorResponse> {
    const path = '/:id';
    return this.client.request<types.ErrorResponse>('PUT', path);
  }

  async delete(): Promise<types.SuccessResponse> {
    const path = '/:id';
    return this.client.request<types.SuccessResponse>('DELETE', path);
  }

  async getByPath(): Promise<types.ErrorResponse> {
    const path = '/path/*path';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getVersions(): Promise<types.ListVersionsResponse> {
    const path = '/:id/versions';
    return this.client.request<types.ListVersionsResponse>('GET', path);
  }

  async rollback(request: types.Rollback_req): Promise<types.ErrorResponse> {
    const path = '/:id/rollback/:version';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getStats(): Promise<void> {
    const path = '/stats';
    return this.client.request<void>('GET', path);
  }

  async getTree(): Promise<void> {
    const path = '/tree';
    return this.client.request<void>('GET', path);
  }

}

export function secretsClient(): SecretsPlugin {
  return new SecretsPlugin();
}
