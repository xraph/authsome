// Auto-generated cms plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class CmsPlugin implements ClientPlugin {
  readonly id = 'cms';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async listEntries(): Promise<void> {
    const path = '/listentries';
    return this.client.request<void>('GET', path);
  }

  async createEntry(): Promise<void> {
    const path = '/createentry';
    return this.client.request<void>('POST', path);
  }

  async getEntry(): Promise<void> {
    const path = '/getentry';
    return this.client.request<void>('GET', path);
  }

  async updateEntry(): Promise<void> {
    const path = '/updateentry';
    return this.client.request<void>('PUT', path);
  }

  async deleteEntry(): Promise<void> {
    const path = '/deleteentry';
    return this.client.request<void>('DELETE', path);
  }

  async publishEntry(): Promise<void> {
    const path = '/publish';
    return this.client.request<void>('POST', path);
  }

  async unpublishEntry(): Promise<void> {
    const path = '/unpublish';
    return this.client.request<void>('POST', path);
  }

  async archiveEntry(): Promise<void> {
    const path = '/archive';
    return this.client.request<void>('POST', path);
  }

  async queryEntries(): Promise<void> {
    const path = '/query';
    return this.client.request<void>('POST', path);
  }

  async bulkPublish(request: types.BulkRequest): Promise<void> {
    const path = '/publish';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async bulkUnpublish(request: types.BulkRequest): Promise<void> {
    const path = '/unpublish';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async bulkDelete(request: types.BulkRequest): Promise<void> {
    const path = '/delete';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getEntryStats(): Promise<void> {
    const path = '/stats';
    return this.client.request<void>('GET', path);
  }

  async listContentTypes(): Promise<void> {
    const path = '/listcontenttypes';
    return this.client.request<void>('GET', path);
  }

  async createContentType(): Promise<void> {
    const path = '/createcontenttype';
    return this.client.request<void>('POST', path);
  }

  async getContentType(): Promise<void> {
    const path = '/:slug';
    return this.client.request<void>('GET', path);
  }

  async updateContentType(): Promise<void> {
    const path = '/:slug';
    return this.client.request<void>('PUT', path);
  }

  async deleteContentType(): Promise<void> {
    const path = '/:slug';
    return this.client.request<void>('DELETE', path);
  }

  async listFields(): Promise<void> {
    const path = '/listfields';
    return this.client.request<void>('GET', path);
  }

  async addField(): Promise<void> {
    const path = '/addfield';
    return this.client.request<void>('POST', path);
  }

  async getField(): Promise<void> {
    const path = '/:fieldSlug';
    return this.client.request<void>('GET', path);
  }

  async updateField(): Promise<void> {
    const path = '/:fieldSlug';
    return this.client.request<void>('PUT', path);
  }

  async deleteField(): Promise<void> {
    const path = '/:fieldSlug';
    return this.client.request<void>('DELETE', path);
  }

  async reorderFields(): Promise<void> {
    const path = '/reorder';
    return this.client.request<void>('POST', path);
  }

  async getFieldTypes(): Promise<void> {
    const path = '/field-types';
    return this.client.request<void>('GET', path);
  }

  async listRevisions(): Promise<void> {
    const path = '/listrevisions';
    return this.client.request<void>('GET', path);
  }

  async getRevision(): Promise<void> {
    const path = '/:version';
    return this.client.request<void>('GET', path);
  }

  async restoreRevision(): Promise<void> {
    const path = '/:version/restore';
    return this.client.request<void>('POST', path);
  }

  async compareRevisions(): Promise<void> {
    const path = '/compare';
    return this.client.request<void>('GET', path);
  }

}

export function cmsClient(): CmsPlugin {
  return new CmsPlugin();
}
