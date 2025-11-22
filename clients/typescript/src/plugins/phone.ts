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

  async sendCode(request: types.SendCodeRequest): Promise<void> {
    const path = '/phone/send-code';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(request: types.VerifyRequest): Promise<types.PhoneVerifyResponse> {
    const path = '/phone/verify';
    return this.client.request<types.PhoneVerifyResponse>('POST', path, {
      body: request,
    });
  }

  async signIn(request: types.VerifyRequest): Promise<types.PhoneVerifyResponse> {
    const path = '/phone/signin';
    return this.client.request<types.PhoneVerifyResponse>('POST', path, {
      body: request,
    });
  }

}

export function phoneClient(): PhonePlugin {
  return new PhonePlugin();
}
