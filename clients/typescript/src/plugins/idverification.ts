// Auto-generated idverification plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class IdverificationPlugin implements ClientPlugin {
  readonly id = 'idverification';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async createVerificationSession(request: types.CreateVerificationSession_req): Promise<types.ErrorResponse> {
    const path = '/sessions';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getVerificationSession(): Promise<types.ErrorResponse> {
    const path = '/sessions/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getVerification(): Promise<types.VerificationResponse> {
    const path = '/:id';
    return this.client.request<types.VerificationResponse>('GET', path);
  }

  async getUserVerifications(): Promise<types.ErrorResponse> {
    const path = '/me';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getUserVerificationStatus(): Promise<types.StatusResponse> {
    const path = '/me/status';
    return this.client.request<types.StatusResponse>('GET', path);
  }

  async requestReverification(request: types.RequestReverification_req): Promise<types.MessageResponse> {
    const path = '/me/reverify';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async handleWebhook(): Promise<types.ErrorResponse> {
    const path = '/webhook/:provider';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

  async adminBlockUser(request: types.AdminBlockUser_req): Promise<types.MessageResponse> {
    const path = '/users/:userId/block';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async adminUnblockUser(): Promise<types.MessageResponse> {
    const path = '/users/:userId/unblock';
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async adminGetUserVerificationStatus(): Promise<types.StatusResponse> {
    const path = '/users/:userId/status';
    return this.client.request<types.StatusResponse>('GET', path);
  }

  async adminGetUserVerifications(): Promise<types.ErrorResponse> {
    const path = '/users/:userId/verifications';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

}

export function idverificationClient(): IdverificationPlugin {
  return new IdverificationPlugin();
}
