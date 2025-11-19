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

  async startRecovery(request: types.StartRecoveryRequest): Promise<types.ErrorResponse> {
    const path = '/recovery/start';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async continueRecovery(request: types.ContinueRecoveryRequest): Promise<types.ErrorResponse> {
    const path = '/recovery/continue';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async completeRecovery(request: types.CompleteRecoveryRequest): Promise<types.ErrorResponse> {
    const path = '/recovery/complete';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async cancelRecovery(request: types.CancelRecoveryRequest): Promise<types.SuccessResponse> {
    const path = '/recovery/cancel';
    return this.client.request<types.SuccessResponse>('POST', path, {
      body: request,
    });
  }

  async generateRecoveryCodes(request: types.GenerateRecoveryCodesRequest): Promise<types.ErrorResponse> {
    const path = '/recovery-codes/generate';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async verifyRecoveryCode(request: types.VerifyRecoveryCodeRequest): Promise<types.ErrorResponse> {
    const path = '/recovery-codes/verify';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async setupSecurityQuestions(request: types.SetupSecurityQuestionsRequest): Promise<types.ErrorResponse> {
    const path = '/security-questions/setup';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getSecurityQuestions(request: types.GetSecurityQuestionsRequest): Promise<types.ErrorResponse> {
    const path = '/security-questions/get';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async verifySecurityAnswers(request: types.VerifySecurityAnswersRequest): Promise<types.ErrorResponse> {
    const path = '/security-questions/verify';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async addTrustedContact(request: types.AddTrustedContactRequest): Promise<types.ErrorResponse> {
    const path = '/trusted-contacts/add';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async listTrustedContacts(): Promise<types.ErrorResponse> {
    const path = '/trusted-contacts';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async verifyTrustedContact(request: types.VerifyTrustedContactRequest): Promise<types.ErrorResponse> {
    const path = '/trusted-contacts/verify';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async requestTrustedContactVerification(request: types.RequestTrustedContactVerificationRequest): Promise<types.ErrorResponse> {
    const path = '/trusted-contacts/request-verification';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async removeTrustedContact(): Promise<types.SuccessResponse> {
    const path = '/trusted-contacts/:id';
    return this.client.request<types.SuccessResponse>('DELETE', path);
  }

  async sendVerificationCode(request: types.SendVerificationCodeRequest): Promise<types.ErrorResponse> {
    const path = '/verification/send';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async verifyCode(request: types.VerifyCodeRequest): Promise<types.ErrorResponse> {
    const path = '/verification/verify';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async scheduleVideoSession(request: types.ScheduleVideoSessionRequest): Promise<types.ErrorResponse> {
    const path = '/video/schedule';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async startVideoSession(request: types.StartVideoSessionRequest): Promise<types.ErrorResponse> {
    const path = '/video/start';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async completeVideoSession(request: types.CompleteVideoSessionRequest): Promise<types.ErrorResponse> {
    const path = '/video/complete';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async uploadDocument(request: types.UploadDocumentRequest): Promise<types.ErrorResponse> {
    const path = '/documents/upload';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getDocumentVerification(): Promise<types.ErrorResponse> {
    const path = '/documents/:id';
    return this.client.request<types.ErrorResponse>('GET', path);
  }

  async reviewDocument(request: types.ReviewDocumentRequest): Promise<types.SuccessResponse> {
    const path = '/documents/:id/review';
    return this.client.request<types.SuccessResponse>('POST', path, {
      body: request,
    });
  }

  async listRecoverySessions(): Promise<void> {
    const path = '/sessions';
    return this.client.request<void>('GET', path);
  }

  async approveRecovery(request: types.ApproveRecoveryRequest): Promise<types.ErrorResponse> {
    const path = '/sessions/:id/approve';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async rejectRecovery(request: types.RejectRecoveryRequest): Promise<types.ErrorResponse> {
    const path = '/sessions/:id/reject';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async getRecoveryStats(): Promise<void> {
    const path = '/stats';
    return this.client.request<void>('GET', path);
  }

  async getRecoveryConfig(): Promise<void> {
    const path = '/config';
    return this.client.request<void>('GET', path);
  }

  async updateRecoveryConfig(request: types.UpdateRecoveryConfigRequest): Promise<types.SuccessResponse> {
    const path = '/config';
    return this.client.request<types.SuccessResponse>('PUT', path, {
      body: request,
    });
  }

  async healthCheck(): Promise<void> {
    const path = '/health';
    return this.client.request<void>('GET', path);
  }

}

export function backupauthClient(): BackupauthPlugin {
  return new BackupauthPlugin();
}
