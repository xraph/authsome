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
    const path = '/cms/listentries';
    return this.client.request<void>('GET', path);
  }

  async createEntry(request: types.CreateEntryRequest): Promise<void> {
    const path = '/cms/createentry';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getEntry(): Promise<void> {
    const path = '/cms/getentry';
    return this.client.request<void>('GET', path);
  }

  async updateEntry(request: types.UpdateEntryRequest): Promise<void> {
    const path = '/cms/updateentry';
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteEntry(): Promise<void> {
    const path = '/cms/deleteentry';
    return this.client.request<void>('DELETE', path);
  }

  async publishEntry(request: types.PublishEntryRequest): Promise<void> {
    const path = '/cms/publish';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async unpublishEntry(): Promise<void> {
    const path = '/cms/unpublish';
    return this.client.request<void>('POST', path);
  }

  async archiveEntry(): Promise<void> {
    const path = '/cms/archive';
    return this.client.request<void>('POST', path);
  }

  async queryEntries(): Promise<void> {
    const path = '/cms/query';
    return this.client.request<void>('POST', path);
  }

  async bulkPublish(): Promise<void> {
    const path = '/cms/publish';
    return this.client.request<void>('POST', path);
  }

  async bulkUnpublish(): Promise<void> {
    const path = '/cms/unpublish';
    return this.client.request<void>('POST', path);
  }

  async bulkDelete(): Promise<void> {
    const path = '/cms/delete';
    return this.client.request<void>('POST', path);
  }

  async getEntryStats(): Promise<void> {
    const path = '/cms/stats';
    return this.client.request<void>('GET', path);
  }

  async listContentTypes(): Promise<void> {
    const path = '/cms/listcontenttypes';
    return this.client.request<void>('GET', path);
  }

  async createContentType(request: types.CreateContentTypeRequest): Promise<void> {
    const path = '/cms/createcontenttype';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getContentType(params: { slug: string }): Promise<void> {
    const path = `/cms/${params.slug}`;
    return this.client.request<void>('GET', path);
  }

  async updateContentType(params: { slug: string }, request: types.UpdateContentTypeRequest): Promise<void> {
    const path = `/cms/${params.slug}`;
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteContentType(params: { slug: string }): Promise<void> {
    const path = `/cms/${params.slug}`;
    return this.client.request<void>('DELETE', path);
  }

  async listFields(): Promise<void> {
    const path = '/cms/listfields';
    return this.client.request<void>('GET', path);
  }

  async addField(request: types.CreateFieldRequest): Promise<void> {
    const path = '/cms/addfield';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getField(params: { fieldSlug: string }): Promise<void> {
    const path = `/cms/${params.fieldSlug}`;
    return this.client.request<void>('GET', path);
  }

  async updateField(params: { fieldSlug: string }, request: types.UpdateFieldRequest): Promise<void> {
    const path = `/cms/${params.fieldSlug}`;
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteField(params: { fieldSlug: string }): Promise<void> {
    const path = `/cms/${params.fieldSlug}`;
    return this.client.request<void>('DELETE', path);
  }

  async reorderFields(request: types.ReorderFieldsRequest): Promise<void> {
    const path = '/cms/reorder';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getFieldTypes(): Promise<void> {
    const path = '/cms/field-types';
    return this.client.request<void>('GET', path);
  }

  async listRevisions(): Promise<void> {
    const path = '/cms/listrevisions';
    return this.client.request<void>('GET', path);
  }

  async getRevision(params: { version: number }): Promise<void> {
    const path = `/cms/${params.version}`;
    return this.client.request<void>('GET', path);
  }

  async restoreRevision(params: { version: number }): Promise<void> {
    const path = `/cms/${params.version}/restore`;
    return this.client.request<void>('POST', path);
  }

  async compareRevisions(): Promise<void> {
    const path = '/cms/compare';
    return this.client.request<void>('GET', path);
  }

}

export function cmsClient(): CmsPlugin {
  return new CmsPlugin();
}
