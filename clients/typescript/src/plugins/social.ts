// Auto-generated social plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class SocialPlugin implements ClientPlugin {
  readonly id = 'social';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async signIn(request: types.SignInRequest): Promise<types.AuthURLResponse> {
    const path = '/signin/social';
    return this.client.request<types.AuthURLResponse>('POST', path, {
      body: request,
    });
  }

  async callback(): Promise<types.CallbackDataResponse> {
    const path = '/callback/:provider';
    return this.client.request<types.CallbackDataResponse>('GET', path);
  }

  async linkAccount(request: types.LinkAccountRequest): Promise<types.AuthURLResponse> {
    const path = '/account/link';
    return this.client.request<types.AuthURLResponse>('POST', path, {
      body: request,
    });
  }

  async unlinkAccount(): Promise<types.MessageResponse> {
    const path = '/account/unlink/:provider';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async listProviders(): Promise<types.ProvidersResponse> {
    const path = '/providers';
    return this.client.request<types.ProvidersResponse>('GET', path);
  }

}

export function socialClient(): SocialPlugin {
  return new SocialPlugin();
}
