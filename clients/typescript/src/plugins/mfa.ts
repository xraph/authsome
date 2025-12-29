// Auto-generated mfa plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class MfaPlugin implements ClientPlugin {
  readonly id = 'mfa';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async enrollFactor(request: types.FactorEnrollmentRequest): Promise<types.FactorEnrollmentResponse> {
    const path = '/mfa/factors/enroll';
    return this.client.request<types.FactorEnrollmentResponse>('POST', path, {
      body: request,
    });
  }

  async listFactors(): Promise<types.FactorsResponse> {
    const path = '/mfa/factors';
    return this.client.request<types.FactorsResponse>('GET', path);
  }

  async getFactor(params: { id: string }): Promise<types.Factor> {
    const path = `/mfa/factors/${params.id}`;
    return this.client.request<types.Factor>('GET', path);
  }

  async updateFactor(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/mfa/factors/${params.id}`;
    return this.client.request<types.MessageResponse>('PUT', path);
  }

  async deleteFactor(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/mfa/factors/${params.id}`;
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async verifyFactor(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/mfa/factors/${params.id}/verify`;
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async initiateChallenge(request: types.ChallengeRequest): Promise<types.ChallengeResponse> {
    const path = '/mfa/challenge';
    return this.client.request<types.ChallengeResponse>('POST', path, {
      body: request,
    });
  }

  async verifyChallenge(request: types.VerificationRequest): Promise<types.VerificationResponse> {
    const path = '/mfa/verify';
    return this.client.request<types.VerificationResponse>('POST', path, {
      body: request,
    });
  }

  async getChallengeStatus(params: { id: string }): Promise<types.ChallengeStatusResponse> {
    const path = `/mfa/challenge/${params.id}`;
    return this.client.request<types.ChallengeStatusResponse>('GET', path);
  }

  async trustDevice(request: types.DeviceInfo): Promise<types.MessageResponse> {
    const path = '/mfa/devices/trust';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async listTrustedDevices(): Promise<types.DevicesResponse> {
    const path = '/mfa/devices';
    return this.client.request<types.DevicesResponse>('GET', path);
  }

  async revokeTrustedDevice(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/mfa/devices/${params.id}`;
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async getStatus(): Promise<types.MFAStatus> {
    const path = '/mfa/status';
    return this.client.request<types.MFAStatus>('GET', path);
  }

  async getPolicy(): Promise<types.MFAConfigResponse> {
    const path = '/mfa/policy';
    return this.client.request<types.MFAConfigResponse>('GET', path);
  }

  async adminUpdatePolicy(): Promise<types.StatusResponse> {
    const path = '/mfa/policy';
    return this.client.request<types.StatusResponse>('PUT', path);
  }

  async adminResetUserMFA(params: { id: string }): Promise<types.MessageResponse> {
    const path = `/mfa/users/${params.id}/reset`;
    return this.client.request<types.MessageResponse>('POST', path);
  }

}

export function mfaClient(): MfaPlugin {
  return new MfaPlugin();
}
