// Auto-generated anonymous plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class AnonymousPlugin implements ClientPlugin {
  readonly id = 'anonymous';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async signIn(): Promise<types.SignInResponse> {
    const path = '/anonymous/signin';
    return this.client.request<types.SignInResponse>('POST', path);
  }

  async link(request: types.LinkRequest): Promise<types.LinkResponse> {
    const path = '/anonymous/link';
    return this.client.request<types.LinkResponse>('POST', path, {
      body: request,
    });
  }

}

export function anonymousClient(): AnonymousPlugin {
  return new AnonymousPlugin();
}
