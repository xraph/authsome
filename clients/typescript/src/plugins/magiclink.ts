// Auto-generated magiclink plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class MagiclinkPlugin implements ClientPlugin {
  readonly id = 'magiclink';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async send(request: types.Send_body): Promise<void> {
    const path = '/magic-link/send';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(): Promise<types.VerifyResponse> {
    const path = '/magic-link/verify';
    return this.client.request<types.VerifyResponse>('GET', path);
  }

}

export function magiclinkClient(): MagiclinkPlugin {
  return new MagiclinkPlugin();
}
