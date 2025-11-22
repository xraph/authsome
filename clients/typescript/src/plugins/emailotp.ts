// Auto-generated emailotp plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class EmailotpPlugin implements ClientPlugin {
  readonly id = 'emailotp';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async send(request: types.SendRequest): Promise<void> {
    const path = '/email-otp/send';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(request: types.VerifyRequest): Promise<types.VerifyResponse> {
    const path = '/email-otp/verify';
    return this.client.request<types.VerifyResponse>('POST', path, {
      body: request,
    });
  }

}

export function emailotpClient(): EmailotpPlugin {
  return new EmailotpPlugin();
}
