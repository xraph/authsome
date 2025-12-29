// Auto-generated backupauth plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class BackupauthPlugin implements ClientPlugin {
  readonly id = 'backupauth';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async startRecovery(request: types.StartRecoveryRequest): Promise<types.BackupAuthRecoveryResponse> {
    const path = '/admin/recovery/start';
    return this.client.request<types.BackupAuthRecoveryResponse>('POST', path, {
      body: request,
    });
  }

  async continueRecovery(request: types.ContinueRecoveryRequest): Promise<types.BackupAuthRecoveryResponse> {
    const path = '/admin/recovery/continue';
    return this.client.request<types.BackupAuthRecoveryResponse>('POST', path, {
      body: request,
    });
  }

  async completeRecovery(request: types.CompleteRecoveryRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/recovery/complete';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async cancelRecovery(request: types.CancelRecoveryRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/recovery/cancel';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async generateRecoveryCodes(request: types.GenerateRecoveryCodesRequest): Promise<types.BackupAuthCodesResponse> {
    const path = '/admin/recovery-codes/generate';
    return this.client.request<types.BackupAuthCodesResponse>('POST', path, {
      body: request,
    });
  }

  async verifyRecoveryCode(request: types.VerifyRecoveryCodeRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/recovery-codes/verify';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async setupSecurityQuestions(request: types.SetupSecurityQuestionsRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/security-questions/setup';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async getSecurityQuestions(request: types.GetSecurityQuestionsRequest): Promise<types.BackupAuthQuestionsResponse> {
    const path = '/admin/security-questions/get';
    return this.client.request<types.BackupAuthQuestionsResponse>('POST', path, {
      body: request,
    });
  }

  async verifySecurityAnswers(request: types.VerifySecurityAnswersRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/security-questions/verify';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async addTrustedContact(request: types.AddTrustedContactRequest): Promise<types.BackupAuthContactResponse> {
    const path = '/admin/trusted-contacts/add';
    return this.client.request<types.BackupAuthContactResponse>('POST', path, {
      body: request,
    });
  }

  async listTrustedContacts(): Promise<types.BackupAuthContactsResponse> {
    const path = '/admin/trusted-contacts';
    return this.client.request<types.BackupAuthContactsResponse>('GET', path);
  }

  async verifyTrustedContact(request: types.VerifyTrustedContactRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/trusted-contacts/verify';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async requestTrustedContactVerification(request: types.RequestTrustedContactVerificationRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/trusted-contacts/request-verification';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async removeTrustedContact(params: { id: string }): Promise<types.BackupAuthStatusResponse> {
    const path = `/admin/trusted-contacts/${params.id}`;
    return this.client.request<types.BackupAuthStatusResponse>('DELETE', path);
  }

  async sendVerificationCode(request: types.SendVerificationCodeRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/verification/send';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async verifyCode(request: types.VerifyCodeRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/verification/verify';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async scheduleVideoSession(request: types.ScheduleVideoSessionRequest): Promise<types.BackupAuthVideoResponse> {
    const path = '/admin/video/schedule';
    return this.client.request<types.BackupAuthVideoResponse>('POST', path, {
      body: request,
    });
  }

  async startVideoSession(request: types.StartVideoSessionRequest): Promise<types.BackupAuthVideoResponse> {
    const path = '/admin/video/start';
    return this.client.request<types.BackupAuthVideoResponse>('POST', path, {
      body: request,
    });
  }

  async completeVideoSession(request: types.CompleteVideoSessionRequest): Promise<types.BackupAuthStatusResponse> {
    const path = '/admin/video/complete';
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async uploadDocument(request: types.UploadDocumentRequest): Promise<types.BackupAuthDocumentResponse> {
    const path = '/admin/documents/upload';
    return this.client.request<types.BackupAuthDocumentResponse>('POST', path, {
      body: request,
    });
  }

  async getDocumentVerification(params: { id: string }): Promise<types.BackupAuthDocumentResponse> {
    const path = `/admin/documents/${params.id}`;
    return this.client.request<types.BackupAuthDocumentResponse>('GET', path);
  }

  async reviewDocument(params: { id: string }, request: types.ReviewDocumentRequest): Promise<types.BackupAuthStatusResponse> {
    const path = `/admin/documents/${params.id}/review`;
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async listRecoverySessions(): Promise<types.BackupAuthSessionsResponse> {
    const path = '/admin/sessions';
    return this.client.request<types.BackupAuthSessionsResponse>('GET', path);
  }

  async approveRecovery(params: { id: string }, request: types.ApproveRecoveryRequest): Promise<types.BackupAuthStatusResponse> {
    const path = `/admin/sessions/${params.id}/approve`;
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async rejectRecovery(params: { id: string }, request: types.RejectRecoveryRequest): Promise<types.BackupAuthStatusResponse> {
    const path = `/admin/sessions/${params.id}/reject`;
    return this.client.request<types.BackupAuthStatusResponse>('POST', path, {
      body: request,
    });
  }

  async getRecoveryStats(): Promise<types.BackupAuthStatsResponse> {
    const path = '/admin/stats';
    return this.client.request<types.BackupAuthStatsResponse>('GET', path);
  }

  async getRecoveryConfig(): Promise<types.BackupAuthConfigResponse> {
    const path = '/admin/config';
    return this.client.request<types.BackupAuthConfigResponse>('GET', path);
  }

  async updateRecoveryConfig(request: types.UpdateRecoveryConfigRequest): Promise<types.BackupAuthConfigResponse> {
    const path = '/admin/config';
    return this.client.request<types.BackupAuthConfigResponse>('PUT', path, {
      body: request,
    });
  }

  async healthCheck(): Promise<void> {
    const path = '/admin/health';
    return this.client.request<void>('GET', path);
  }

}

export function backupauthClient(): BackupauthPlugin {
  return new BackupauthPlugin();
}
