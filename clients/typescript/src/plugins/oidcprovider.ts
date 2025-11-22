// Auto-generated oidcprovider plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class OidcproviderPlugin implements ClientPlugin {
  readonly id = 'oidcprovider';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async registerClient(request: types.ClientRegistrationRequest): Promise<void> {
    const path = '/register';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async listClients(): Promise<void> {
    const path = '/listclients';
    return this.client.request<void>('GET', path);
  }

  async getClient(): Promise<void> {
    const path = '/:clientId';
    return this.client.request<void>('GET', path);
  }

  async updateClient(request: types.ClientUpdateRequest): Promise<void> {
    const path = '/:clientId';
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteClient(): Promise<void> {
    const path = '/:clientId';
    return this.client.request<void>('DELETE', path);
  }

  async discovery(): Promise<void> {
    const path = '/.well-known/openid-configuration';
    return this.client.request<void>('GET', path);
  }

  async jWKS(): Promise<void> {
    const path = '/jwks';
    return this.client.request<void>('GET', path);
  }

  async authorize(): Promise<void> {
    const path = '/authorize';
    return this.client.request<void>('GET', path);
  }

  async handleConsent(): Promise<void> {
    const path = '/consent';
    return this.client.request<void>('POST', path);
  }

  async token(request: types.TokenRequest): Promise<void> {
    const path = '/token';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async userInfo(): Promise<void> {
    const path = '/userinfo';
    return this.client.request<void>('GET', path);
  }

  async introspectToken(): Promise<void> {
    const path = '/introspect';
    return this.client.request<void>('POST', path);
  }

  async revokeToken(): Promise<types.StatusResponse> {
    const path = '/revoke';
    return this.client.request<types.StatusResponse>('POST', path);
  }

}

export function oidcproviderClient(): OidcproviderPlugin {
  return new OidcproviderPlugin();
}
