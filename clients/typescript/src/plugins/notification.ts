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

  async createTemplate(request: types.CreateTemplateRequest): Promise<types.NotificationTemplateResponse> {
    const path = '/templates';
    return this.client.request<types.NotificationTemplateResponse>('POST', path, {
      body: request,
    });
  }

  async getTemplate(params: { id: string }): Promise<types.NotificationTemplateResponse> {
    const path = `/templates/${params.id}`;
    return this.client.request<types.NotificationTemplateResponse>('GET', path);
  }

  async listTemplates(): Promise<types.NotificationTemplateListResponse> {
    const path = '/templates';
    return this.client.request<types.NotificationTemplateListResponse>('GET', path);
  }

  async updateTemplate(params: { id: string }, request: types.UpdateTemplateRequest): Promise<types.NotificationTemplateResponse> {
    const path = `/templates/${params.id}`;
    return this.client.request<types.NotificationTemplateResponse>('PUT', path, {
      body: request,
    });
  }

  async deleteTemplate(params: { id: string }): Promise<types.NotificationStatusResponse> {
    const path = `/templates/${params.id}`;
    return this.client.request<types.NotificationStatusResponse>('DELETE', path);
  }

  async resetTemplate(params: { id: string }): Promise<types.NotificationStatusResponse> {
    const path = `/templates/${params.id}/reset`;
    return this.client.request<types.NotificationStatusResponse>('POST', path);
  }

  async resetAllTemplates(): Promise<types.NotificationStatusResponse> {
    const path = '/templates/reset-all';
    return this.client.request<types.NotificationStatusResponse>('POST', path);
  }

  async getTemplateDefaults(): Promise<types.NotificationTemplateListResponse> {
    const path = '/templates/defaults';
    return this.client.request<types.NotificationTemplateListResponse>('GET', path);
  }

  async previewTemplate(params: { id: string }, request: types.PreviewTemplate_req): Promise<types.NotificationPreviewResponse> {
    const path = `/templates/${params.id}/preview`;
    return this.client.request<types.NotificationPreviewResponse>('POST', path, {
      body: request,
    });
  }

  async renderTemplate(request: types.RenderTemplate_req): Promise<types.NotificationPreviewResponse> {
    const path = '/templates/render';
    return this.client.request<types.NotificationPreviewResponse>('POST', path, {
      body: request,
    });
  }

  async sendNotification(request: types.SendRequest): Promise<types.NotificationResponse> {
    const path = '/notifications/send';
    return this.client.request<types.NotificationResponse>('POST', path, {
      body: request,
    });
  }

  async getNotification(params: { id: string }): Promise<types.NotificationResponse> {
    const path = `/notifications/${params.id}`;
    return this.client.request<types.NotificationResponse>('GET', path);
  }

  async listNotifications(): Promise<types.NotificationListResponse> {
    const path = '/notifications';
    return this.client.request<types.NotificationListResponse>('GET', path);
  }

  async resendNotification(params: { id: string }): Promise<types.NotificationResponse> {
    const path = `/notifications/${params.id}/resend`;
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
