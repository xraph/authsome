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

  async list(request?: types.ListSecretsRequest): Promise<types.ListSecretsResponse> {
    const path = '/secrets/list';
    return this.client.request<types.ListSecretsResponse>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async create(request: types.CreateSecretRequest): Promise<types.SecretDTO> {
    const path = '/secrets/create';
    return this.client.request<types.SecretDTO>('POST', path, {
      body: request,
    });
  }

  async get(params: { id: string }, request?: types.GetSecretRequest): Promise<types.SecretDTO> {
    const path = `/secrets/${params.id}`;
    return this.client.request<types.SecretDTO>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async getValue(params: { id: string }, request?: types.GetValueRequest): Promise<types.RevealValueResponse> {
    const path = `/secrets/${params.id}/value`;
    return this.client.request<types.RevealValueResponse>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async update(params: { id: string }, request: types.UpdateSecretRequest): Promise<types.SecretDTO> {
    const path = `/secrets/${params.id}`;
    return this.client.request<types.SecretDTO>('PUT', path, {
      body: request,
    });
  }

  async delete(params: { id: string }, request?: types.DeleteSecretRequest): Promise<types.SuccessResponse> {
    const path = `/secrets/${params.id}`;
    return this.client.request<types.SuccessResponse>('DELETE', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async getByPath(): Promise<types.ErrorResponse> {
    const path = '/secrets/path/*path';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getVersions(params: { id: string }, request?: types.GetVersionsRequest): Promise<types.ListVersionsResponse> {
    const path = `/secrets/${params.id}/versions`;
    return this.client.request<types.ListVersionsResponse>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async rollback(params: { id: string; version: number }, request: types.RollbackRequest): Promise<types.SecretDTO> {
    const path = `/secrets/${params.id}/rollback/${params.version}`;
    return this.client.request<types.SecretDTO>('POST', path, {
      body: request,
    });
  }

  async getStats(): Promise<types.StatsDTO> {
    const path = '/secrets/stats';
    return this.client.request<types.StatsDTO>('GET', path);
  }

  async getTree(request?: types.GetTreeRequest): Promise<types.SecretTreeNode> {
    const path = '/secrets/tree';
    return this.client.request<types.SecretTreeNode>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

}

export function secretsClient(): SecretsPlugin {
  return new SecretsPlugin();
}
