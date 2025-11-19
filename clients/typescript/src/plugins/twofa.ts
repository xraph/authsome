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

  async enable(request: types.Enable_body): Promise<void> {
    const path = '/2fa/enable';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async verify(request: types.Verify_body): Promise<types.StatusResponse> {
    const path = '/2fa/verify';
    return this.client.request<types.StatusResponse>('POST', path, {
      body: request,
    });
  }

  async disable(request: types.Disable_body): Promise<types.StatusResponse> {
    const path = '/2fa/disable';
    return this.client.request<types.StatusResponse>('POST', path, {
      body: request,
    });
  }

  async generateBackupCodes(request: types.GenerateBackupCodes_body): Promise<types.CodesResponse> {
    const path = '/2fa/generate-backup-codes';
    return this.client.request<types.CodesResponse>('POST', path, {
      body: request,
    });
  }

  async sendOTP(request: types.SendOTP_body): Promise<types.OTPSentResponse> {
    const path = '/2fa/send-otp';
    return this.client.request<types.OTPSentResponse>('POST', path, {
      body: request,
    });
  }

  async status(request: types.Status_body): Promise<types.TwoFAStatusResponse> {
    const path = '/2fa/status';
    return this.client.request<types.TwoFAStatusResponse>('POST', path, {
      body: request,
    });
  }

}

export function twofaClient(): TwofaPlugin {
  return new TwofaPlugin();
}
