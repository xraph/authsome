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

  async createVerificationSession(request: types.CreateVerificationSession_req): Promise<types.IDVerificationSessionResponse> {
    const path = '/verification/sessions';
    return this.client.request<types.IDVerificationSessionResponse>('POST', path, {
      body: request,
    });
  }

  async getVerificationSession(params: { id: string }): Promise<types.IDVerificationSessionResponse> {
    const path = `/verification/sessions/${params.id}`;
    return this.client.request<types.IDVerificationSessionResponse>('GET', path);
  }

  async getVerification(params: { id: string }): Promise<types.IDVerificationResponse> {
    const path = `/verification/${params.id}`;
    return this.client.request<types.IDVerificationResponse>('GET', path);
  }

  async getUserVerifications(): Promise<types.IDVerificationListResponse> {
    const path = '/verification/me';
    return this.client.request<types.IDVerificationListResponse>('GET', path);
  }

  async getUserVerificationStatus(): Promise<types.IDVerificationStatusResponse> {
    const path = '/verification/me/status';
    return this.client.request<types.IDVerificationStatusResponse>('GET', path);
  }

  async requestReverification(request: types.RequestReverification_req): Promise<types.IDVerificationSessionResponse> {
    const path = '/verification/me/reverify';
    return this.client.request<types.IDVerificationSessionResponse>('POST', path, {
      body: request,
    });
  }

  async handleWebhook(params: { provider: string }): Promise<types.IDVerificationWebhookResponse> {
    const path = `/verification/webhook/${params.provider}`;
    return this.client.request<types.IDVerificationWebhookResponse>('POST', path);
  }

  async adminBlockUser(params: { userId: string }, request: types.AdminBlockUser_req): Promise<types.IDVerificationStatusResponse> {
    const path = `/verification/admin/users/${params.userId}/block`;
    return this.client.request<types.IDVerificationStatusResponse>('POST', path, {
      body: request,
    });
  }

  async adminUnblockUser(params: { userId: string }): Promise<types.IDVerificationStatusResponse> {
    const path = `/verification/admin/users/${params.userId}/unblock`;
    return this.client.request<types.IDVerificationStatusResponse>('POST', path);
  }

  async adminGetUserVerificationStatus(params: { userId: string }): Promise<types.IDVerificationStatusResponse> {
    const path = `/verification/admin/users/${params.userId}/status`;
    return this.client.request<types.IDVerificationStatusResponse>('GET', path);
  }

  async adminGetUserVerifications(params: { userId: string }): Promise<types.IDVerificationListResponse> {
    const path = `/verification/admin/users/${params.userId}/verifications`;
    return this.client.request<types.IDVerificationListResponse>('GET', path);
  }

}

export function idverificationClient(): IdverificationPlugin {
  return new IdverificationPlugin();
}
