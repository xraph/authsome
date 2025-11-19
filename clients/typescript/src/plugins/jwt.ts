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

  async createJWTKey(): Promise<void> {
    const path = '/createjwtkey';
    return this.client.request<void>('POST', path);
  }

  async listJWTKeys(): Promise<void> {
    const path = '/listjwtkeys';
    return this.client.request<void>('GET', path);
  }

  async getJWKS(): Promise<void> {
    const path = '/jwks';
    return this.client.request<void>('GET', path);
  }

  async generateToken(): Promise<void> {
    const path = '/generate';
    return this.client.request<void>('POST', path);
  }

  async verifyToken(): Promise<void> {
    const path = '/verify';
    return this.client.request<void>('POST', path);
  }

}

export function jwtClient(): JwtPlugin {
  return new JwtPlugin();
}
