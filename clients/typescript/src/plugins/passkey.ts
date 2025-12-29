// Auto-generated passkey plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class PasskeyPlugin implements ClientPlugin {
  readonly id = 'passkey';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async beginRegister(): Promise<void> {
    const path = '/passkey/register/begin';
    return this.client.request<void>('POST', path);
  }

  async finishRegister(): Promise<void> {
    const path = '/passkey/register/finish';
    return this.client.request<void>('POST', path);
  }

  async beginLogin(): Promise<void> {
    const path = '/passkey/login/begin';
    return this.client.request<void>('POST', path);
  }

  async finishLogin(): Promise<void> {
    const path = '/passkey/login/finish';
    return this.client.request<void>('POST', path);
  }

  async list(): Promise<void> {
    const path = '/passkey/list';
    return this.client.request<void>('GET', path);
  }

  async update(params: { id: string }): Promise<void> {
    const path = `/passkey/${params.id}`;
    return this.client.request<void>('PUT', path);
  }

  async delete(params: { id: string }): Promise<types.StatusResponse> {
    const path = `/passkey/${params.id}`;
    return this.client.request<types.StatusResponse>('DELETE', path);
  }

}

export function passkeyClient(): PasskeyPlugin {
  return new PasskeyPlugin();
}
