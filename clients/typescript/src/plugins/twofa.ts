// Auto-generated twofa plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class TwofaPlugin implements ClientPlugin {
  readonly id = 'twofa';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async enable(): Promise<void> {
    const path = '/2fa/enable';
    return this.client.request<void>('POST', path);
  }

  async verify(): Promise<types.StatusResponse> {
    const path = '/2fa/verify';
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async disable(): Promise<types.StatusResponse> {
    const path = '/2fa/disable';
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async generateBackupCodes(): Promise<types.CodesResponse> {
    const path = '/2fa/generate-backup-codes';
    return this.client.request<types.CodesResponse>('POST', path);
  }

  async sendOTP(): Promise<types.OTPSentResponse> {
    const path = '/2fa/send-otp';
    return this.client.request<types.OTPSentResponse>('POST', path);
  }

  async status(): Promise<types.TwoFAStatusResponse> {
    const path = '/2fa/status';
    return this.client.request<types.TwoFAStatusResponse>('POST', path);
  }

}

export function twofaClient(): TwofaPlugin {
  return new TwofaPlugin();
}
