// Auto-generated webhook plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class WebhookPlugin implements ClientPlugin {
  readonly id = 'webhook';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async create(request: { events: string[]; secret?: string; url: string }): Promise<{ webhook: types.Webhook }> {
    const path = '/webhooks';
    return this.client.request<{ webhook: types.Webhook }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async list(): Promise<{ webhooks: types.Webhook[] }> {
    const path = '/webhooks';
    return this.client.request<{ webhooks: types.Webhook[] }>('GET', path, {
      auth: true,
    });
  }

  async update(request: { events?: string[]; enabled?: boolean; id: string; url?: string }): Promise<{ webhook: types.Webhook }> {
    const path = '/webhooks/update';
    return this.client.request<{ webhook: types.Webhook }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async delete(request: { id: string }): Promise<{ success: boolean }> {
    const path = '/webhooks/delete';
    return this.client.request<{ success: boolean }>('POST', path, {
      body: request,
      auth: true,
    });
  }

}

export function webhookClient(): WebhookPlugin {
  return new WebhookPlugin();
}
