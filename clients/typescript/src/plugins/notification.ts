// Auto-generated notification plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class NotificationPlugin implements ClientPlugin {
  readonly id = 'notification';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async previewTemplate(params: { id: string }, request: types.PreviewTemplate_req): Promise<types.NotificationPreviewResponse> {
    const path = `/${params.id}/preview`;
    return this.client.request<types.NotificationPreviewResponse>('POST', path, {
      body: request,
    });
  }

  async createTemplate(request: types.CreateTemplateRequest): Promise<types.NotificationTemplateResponse> {
    const path = '/createtemplate';
    return this.client.request<types.NotificationTemplateResponse>('POST', path, {
      body: request,
    });
  }

  async getTemplate(params: { id: string }): Promise<types.NotificationTemplateResponse> {
    const path = `/${params.id}`;
    return this.client.request<types.NotificationTemplateResponse>('GET', path);
  }

  async listTemplates(): Promise<types.NotificationTemplateListResponse> {
    const path = '/listtemplates';
    return this.client.request<types.NotificationTemplateListResponse>('GET', path);
  }

  async updateTemplate(params: { id: string }, request: types.UpdateTemplateRequest): Promise<types.NotificationTemplateResponse> {
    const path = `/${params.id}`;
    return this.client.request<types.NotificationTemplateResponse>('PUT', path, {
      body: request,
    });
  }

  async deleteTemplate(params: { id: string }): Promise<types.NotificationStatusResponse> {
    const path = `/${params.id}`;
    return this.client.request<types.NotificationStatusResponse>('DELETE', path);
  }

  async resetTemplate(params: { id: string }): Promise<types.NotificationStatusResponse> {
    const path = `/${params.id}/reset`;
    return this.client.request<types.NotificationStatusResponse>('POST', path);
  }

  async resetAllTemplates(): Promise<types.NotificationStatusResponse> {
    const path = '/reset-all';
    return this.client.request<types.NotificationStatusResponse>('POST', path);
  }

  async getTemplateDefaults(): Promise<types.NotificationTemplateListResponse> {
    const path = '/defaults';
    return this.client.request<types.NotificationTemplateListResponse>('GET', path);
  }

  async renderTemplate(request: types.RenderTemplate_req): Promise<types.NotificationPreviewResponse> {
    const path = '/render';
    return this.client.request<types.NotificationPreviewResponse>('POST', path, {
      body: request,
    });
  }

  async sendNotification(request: types.SendRequest): Promise<types.NotificationResponse> {
    const path = '/send';
    return this.client.request<types.NotificationResponse>('POST', path, {
      body: request,
    });
  }

  async getNotification(params: { id: string }): Promise<types.NotificationResponse> {
    const path = `/${params.id}`;
    return this.client.request<types.NotificationResponse>('GET', path);
  }

  async listNotifications(): Promise<types.NotificationListResponse> {
    const path = '/listnotifications';
    return this.client.request<types.NotificationListResponse>('GET', path);
  }

  async resendNotification(params: { id: string }): Promise<types.NotificationResponse> {
    const path = `/${params.id}/resend`;
    return this.client.request<types.NotificationResponse>('POST', path);
  }

  async handleWebhook(params: { provider: string }): Promise<types.NotificationWebhookResponse> {
    const path = `/notifications/webhook/${params.provider}`;
    return this.client.request<types.NotificationWebhookResponse>('POST', path);
  }

}

export function notificationClient(): NotificationPlugin {
  return new NotificationPlugin();
}
