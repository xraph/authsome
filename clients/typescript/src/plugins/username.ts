// Auto-generated username plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class UsernamePlugin implements ClientPlugin {
  readonly id = 'username';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async signUp(request: types.SignUpRequest): Promise<types.SignUpResponse> {
    const path = '/username/signup';
    return this.client.request<types.SignUpResponse>('POST', path, {
      body: request,
    });
  }

  async signIn(request: types.SignInRequest): Promise<types.TwoFARequiredResponse> {
    const path = '/username/signin';
    return this.client.request<types.TwoFARequiredResponse>('POST', path, {
      body: request,
    });
  }

}

export function usernameClient(): UsernamePlugin {
  return new UsernamePlugin();
}
