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

  async createVerificationSession(request: types.CreateVerificationSession_req): Promise<types.VerificationSessionResponse> {
    const path = '/sessions';
    return this.client.request<types.VerificationSessionResponse>('POST', path, {
      body: request,
    });
  }

  async getVerificationSession(): Promise<types.VerificationSessionResponse> {
    const path = '/sessions/:id';
    return this.client.request<types.VerificationSessionResponse>('GET', path);
  }

  async getVerification(): Promise<types.VerificationResponse> {
    const path = '/:id';
    return this.client.request<types.VerificationResponse>('GET', path);
  }

  async getUserVerifications(): Promise<types.VerificationListResponse> {
    const path = '/me';
    return this.client.request<types.VerificationListResponse>('GET', path);
  }

  async getUserVerificationStatus(): Promise<types.UserVerificationStatusResponse> {
    const path = '/me/status';
    return this.client.request<types.UserVerificationStatusResponse>('GET', path);
  }

  async requestReverification(request: types.RequestReverification_req): Promise<types.MessageResponse> {
    const path = '/me/reverify';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async handleWebhook(): Promise<void> {
    const path = '/webhook/:provider';
    return this.client.request<void>('POST', path);
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

  async adminGetUserVerificationStatus(): Promise<types.UserVerificationStatusResponse> {
    const path = '/users/:userId/status';
    return this.client.request<types.UserVerificationStatusResponse>('GET', path);
  }

  async adminGetUserVerifications(): Promise<types.VerificationListResponse> {
    const path = '/users/:userId/verifications';
    return this.client.request<types.VerificationListResponse>('GET', path);
  }

}

export function idverificationClient(): IdverificationPlugin {
  return new IdverificationPlugin();
}
