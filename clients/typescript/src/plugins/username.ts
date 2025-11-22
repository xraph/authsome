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

  async signUp(): Promise<types.SignUpResponse> {
    const path = '/username/signup';
    return this.client.request<types.SignUpResponse>('POST', path);
  }

  async signIn(): Promise<types.SignInResponse> {
    const path = '/username/signin';
    return this.client.request<types.SignInResponse>('POST', path);
  }

}

export function usernameClient(): UsernamePlugin {
  return new UsernamePlugin();
}
