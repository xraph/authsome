// Auto-generated jwt plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class JwtPlugin implements ClientPlugin {
  readonly id = 'jwt';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createJWTKey(request: types.CreateJWTKeyRequest): Promise<types.JWTKey> {
    const path = '/jwt/keys';
    return this.client.request<types.JWTKey>('POST', path, {
      body: request,
    });
  }

  async listJWTKeys(request?: types.ListJWTKeysRequest): Promise<types.ListJWTKeysResponse> {
    const path = '/jwt/keys';
    return this.client.request<types.ListJWTKeysResponse>('GET', path, {
      query: this.client.toQueryParams(request),
    });
  }

  async getJWKS(): Promise<types.JWKSResponse> {
    const path = '/jwt/jwks';
    return this.client.request<types.JWKSResponse>('GET', path);
  }

  async generateToken(request: types.GenerateTokenRequest): Promise<types.GenerateTokenResponse> {
    const path = '/jwt/generate';
    return this.client.request<types.GenerateTokenResponse>('POST', path, {
      body: request,
    });
  }

  async verifyToken(request: types.VerifyTokenRequest): Promise<types.VerifyTokenResponse> {
    const path = '/jwt/verify';
    return this.client.request<types.VerifyTokenResponse>('POST', path, {
      body: request,
    });
  }

}

export function jwtClient(): JwtPlugin {
  return new JwtPlugin();
}
