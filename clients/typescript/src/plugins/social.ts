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

  async callback(params: { provider: string }, query?: { state?: string; code?: string; error?: string; errorDescription?: string }): Promise<types.CallbackDataResponse> {
    const path = `/callback/${params.provider}`;
    return this.client.request<types.CallbackDataResponse>('GET', path, {
      query: this.client.toQueryParams(query),
    });
  }

  async linkAccount(request: types.LinkAccountRequest): Promise<types.AuthURLResponse> {
    const path = '/account/link';
    return this.client.request<types.AuthURLResponse>('POST', path, {
      body: request,
    });
  }

  async unlinkAccount(params: { provider: string }): Promise<types.MessageResponse> {
    const path = `/account/unlink/${params.provider}`;
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
