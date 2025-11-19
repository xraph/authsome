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

  async authorize(): Promise<void> {
    const path = '/authorize';
    return this.client.request<void>('GET', path);
  }

  async token(request: types.TokenRequest): Promise<types.ErrorResponse> {
    const path = '/token';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async userInfo(): Promise<types.ErrorResponse> {
    const path = '/userinfo';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async jWKS(): Promise<types.ErrorResponse> {
    const path = '/jwks';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async registerClient(request: types.RegisterClient_req): Promise<types.ErrorResponse> {
    const path = '/register';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async handleConsent(): Promise<types.ErrorResponse> {
    const path = '/consent';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

}

export function oidcproviderClient(): OidcproviderPlugin {
  return new OidcproviderPlugin();
}
