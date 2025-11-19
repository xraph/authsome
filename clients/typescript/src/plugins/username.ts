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

  async signUp(request: types.SignUp_body): Promise<types.StatusResponse> {
    const path = '/username/signup';
    return this.client.request<types.StatusResponse>('POST', path, {
      body: request,
    });
  }

  async signIn(request: types.SignIn_body): Promise<types.SignInResponse> {
    const path = '/username/signin';
    return this.client.request<types.SignInResponse>('POST', path, {
      body: request,
    });
  }

}

export function usernameClient(): UsernamePlugin {
  return new UsernamePlugin();
}
