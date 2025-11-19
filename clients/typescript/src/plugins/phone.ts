// Auto-generated phone plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class PhonePlugin implements ClientPlugin {
  readonly id = 'phone';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async sendCode(request: types.SendCode_body): Promise<void> {
    const path = '/phone/send-code';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(request: types.Verify_body): Promise<types.VerifyResponse> {
    const path = '/phone/verify';
    return this.client.request<types.VerifyResponse>('POST', path, {
      body: request,
    });
  }

  async signIn(request: types.SignIn_body): Promise<types.VerifyResponse> {
    const path = '/phone/signin';
    return this.client.request<types.VerifyResponse>('POST', path, {
      body: request,
    });
  }

}

export function phoneClient(): PhonePlugin {
  return new PhonePlugin();
}
