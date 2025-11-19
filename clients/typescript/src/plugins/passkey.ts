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

  async beginRegister(request: types.BeginRegister_body): Promise<void> {
    const path = '/register/begin';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async finishRegister(request: types.FinishRegister_body): Promise<types.StatusResponse> {
    const path = '/register/finish';
    return this.client.request<types.StatusResponse>('POST', path, {
      body: request,
    });
  }

  async beginLogin(request: types.BeginLogin_body): Promise<void> {
    const path = '/login/begin';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async finishLogin(request: types.FinishLogin_body): Promise<types.AuthResponse> {
    const path = '/login/finish';
    return this.client.request<types.AuthResponse>('POST', path, {
      body: request,
    });
  }

  async list(): Promise<void> {
    const path = '/list';
    return this.client.request<void>('GET', path);
  }

  async delete(): Promise<types.StatusResponse> {
    const path = '/:id';
    return this.client.request<types.StatusResponse>('DELETE', path);
  }

}

export function passkeyClient(): PasskeyPlugin {
  return new PasskeyPlugin();
}
