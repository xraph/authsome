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
    const path = '/register/begin';
    return this.client.request<void>('POST', path);
  }

  async finishRegister(): Promise<void> {
    const path = '/register/finish';
    return this.client.request<void>('POST', path);
  }

  async beginLogin(): Promise<void> {
    const path = '/login/begin';
    return this.client.request<void>('POST', path);
  }

  async finishLogin(): Promise<void> {
    const path = '/login/finish';
    return this.client.request<void>('POST', path);
  }

  async list(): Promise<void> {
    const path = '/list';
    return this.client.request<void>('GET', path);
  }

  async update(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('PUT', path);
  }

  async delete(): Promise<types.StatusResponse> {
    const path = '/:id';
    return this.client.request<types.StatusResponse>('DELETE', path);
  }

}

export function passkeyClient(): PasskeyPlugin {
  return new PasskeyPlugin();
}
