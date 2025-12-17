// Auto-generated emailverification plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class EmailverificationPlugin implements ClientPlugin {
  readonly id = 'emailverification';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async send(request: types.SendRequest): Promise<void> {
    const path = '/email-verification/send';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(): Promise<void> {
    const path = '/email-verification/verify';
    return this.client.request<void>('GET', path);
  }

  async resend(request: types.ResendRequest): Promise<types.ResendResponse> {
    const path = '/email-verification/resend';
    return this.client.request<types.ResendResponse>('POST', path, {
      body: request,
    });
  }

}

export function emailverificationClient(): EmailverificationPlugin {
  return new EmailverificationPlugin();
}
