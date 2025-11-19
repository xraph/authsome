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

  async enrollFactor(request: types.FactorEnrollmentRequest): Promise<types.ErrorResponse> {
    const path = '/mfa/factors/enroll';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listFactors(): Promise<types.FactorsResponse> {
    const path = '/mfa/factors';
    return this.client.request<types.FactorsResponse>('GET', path);
  }

  async getFactor(): Promise<types.ErrorResponse> {
    const path = '/mfa/factors/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async updateFactor(): Promise<types.MessageResponse> {
    const path = '/mfa/factors/:id';
    return this.client.request<types.MessageResponse>('PUT', path);
  }

  async deleteFactor(): Promise<types.MessageResponse> {
    const path = '/mfa/factors/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async verifyFactor(request: types.VerifyFactor_req): Promise<types.MessageResponse> {
    const path = '/mfa/factors/:id/verify';
    return this.client.request<types.MessageResponse>('POST', path, {
      body: request,
    });
  }

  async initiateChallenge(request: types.ChallengeRequest): Promise<types.ErrorResponse> {
    const path = '/mfa/challenge';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async verifyChallenge(request: types.VerificationRequest): Promise<types.ErrorResponse> {
    const path = '/mfa/verify';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getChallengeStatus(): Promise<types.ErrorResponse> {
    const path = '/mfa/challenge/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
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

  async revokeTrustedDevice(): Promise<types.MessageResponse> {
    const path = '/mfa/devices/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async getStatus(): Promise<types.ErrorResponse> {
    const path = '/mfa/status';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async getPolicy(): Promise<types.MFAConfigResponse> {
    const path = '/mfa/policy';
    return this.client.request<types.MFAConfigResponse>('GET', path);
  }

}

export function mfaClient(): MfaPlugin {
  return new MfaPlugin();
}
