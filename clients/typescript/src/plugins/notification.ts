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

  async previewTemplate(request: types.PreviewTemplate_req): Promise<void> {
    const path = '/:id/preview';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async createTemplate(): Promise<void> {
    const path = '/createtemplate';
    return this.client.request<void>('POST', path);
  }

  async getTemplate(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('GET', path);
  }

  async listTemplates(): Promise<void> {
    const path = '/listtemplates';
    return this.client.request<void>('GET', path);
  }

  async updateTemplate(): Promise<types.MessageResponse> {
    const path = '/:id';
    return this.client.request<types.MessageResponse>('PUT', path);
  }

  async deleteTemplate(): Promise<types.MessageResponse> {
    const path = '/:id';
    return this.client.request<types.MessageResponse>('DELETE', path);
  }

  async resetTemplate(): Promise<types.MessageResponse> {
    const path = '/:id/reset';
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async resetAllTemplates(): Promise<types.MessageResponse> {
    const path = '/reset-all';
    return this.client.request<types.MessageResponse>('POST', path);
  }

  async getTemplateDefaults(): Promise<void> {
    const path = '/defaults';
    return this.client.request<void>('GET', path);
  }

  async renderTemplate(request: types.RenderTemplate_req): Promise<void> {
    const path = '/render';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async sendNotification(): Promise<void> {
    const path = '/send';
    return this.client.request<void>('POST', path);
  }

  async getNotification(): Promise<void> {
    const path = '/:id';
    return this.client.request<void>('GET', path);
  }

  async listNotifications(): Promise<void> {
    const path = '/listnotifications';
    return this.client.request<void>('GET', path);
  }

  async resendNotification(): Promise<void> {
    const path = '/:id/resend';
    return this.client.request<void>('POST', path);
  }

  async handleWebhook(): Promise<types.StatusResponse> {
    const path = '/notifications/webhook/:provider';
    return this.client.request<types.StatusResponse>('POST', path);
  }

}

export function notificationClient(): NotificationPlugin {
  return new NotificationPlugin();
}
