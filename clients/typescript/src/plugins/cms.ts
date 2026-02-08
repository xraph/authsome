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

  async listEntries(params: { typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}`;
    return this.client.request<void>('GET', path);
  }

  async createEntry(params: { typeSlug: string }, request: types.CreateEntryRequest): Promise<void> {
    const path = `/cms/${params.typeSlug}`;
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getEntry(params: { typeSlug: string; entryId: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}`;
    return this.client.request<void>('GET', path);
  }

  async updateEntry(params: { typeSlug: string; entryId: string }, request: types.UpdateEntryRequest): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}`;
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteEntry(params: { typeSlug: string; entryId: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}`;
    return this.client.request<void>('DELETE', path);
  }

  async publishEntry(params: { typeSlug: string; entryId: string }, request: types.PublishEntryRequest): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/publish`;
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async unpublishEntry(params: { typeSlug: string; entryId: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/unpublish`;
    return this.client.request<void>('POST', path);
  }

  async archiveEntry(params: { typeSlug: string; entryId: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/archive`;
    return this.client.request<void>('POST', path);
  }

  async queryEntries(params: { typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/query`;
    return this.client.request<void>('POST', path);
  }

  async bulkPublish(params: { typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/bulk/publish`;
    return this.client.request<void>('POST', path);
  }

  async bulkUnpublish(params: { typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/bulk/unpublish`;
    return this.client.request<void>('POST', path);
  }

  async bulkDelete(params: { typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/bulk/delete`;
    return this.client.request<void>('POST', path);
  }

  async getEntryStats(params: { typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/stats`;
    return this.client.request<void>('GET', path);
  }

  async listContentTypes(): Promise<void> {
    const path = '/cms/types';
    return this.client.request<void>('GET', path);
  }

  async createContentType(request: types.CreateContentTypeRequest): Promise<void> {
    const path = '/cms/types';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getContentType(params: { slug: string }): Promise<void> {
    const path = `/cms/types/${params.slug}`;
    return this.client.request<void>('GET', path);
  }

  async updateContentType(params: { slug: string }, request: types.UpdateContentTypeRequest): Promise<void> {
    const path = `/cms/types/${params.slug}`;
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteContentType(params: { slug: string }): Promise<void> {
    const path = `/cms/types/${params.slug}`;
    return this.client.request<void>('DELETE', path);
  }

  async listFields(params: { slug: string }): Promise<void> {
    const path = `/cms/types/${params.slug}/fields`;
    return this.client.request<void>('GET', path);
  }

  async addField(params: { slug: string }, request: types.CreateFieldRequest): Promise<void> {
    const path = `/cms/types/${params.slug}/fields`;
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getField(params: { slug: string; fieldSlug: string }): Promise<void> {
    const path = `/cms/types/${params.slug}/fields/${params.fieldSlug}`;
    return this.client.request<void>('GET', path);
  }

  async updateField(params: { slug: string; fieldSlug: string }, request: types.UpdateFieldRequest): Promise<void> {
    const path = `/cms/types/${params.slug}/fields/${params.fieldSlug}`;
    return this.client.request<void>('PUT', path, {
      body: request,
    });
  }

  async deleteField(params: { slug: string; fieldSlug: string }): Promise<void> {
    const path = `/cms/types/${params.slug}/fields/${params.fieldSlug}`;
    return this.client.request<void>('DELETE', path);
  }

  async reorderFields(params: { slug: string }, request: types.ReorderFieldsRequest): Promise<void> {
    const path = `/cms/types/${params.slug}/fields/reorder`;
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async getFieldTypes(): Promise<void> {
    const path = '/cms/field-types';
    return this.client.request<void>('GET', path);
  }

  async listRevisions(params: { typeSlug: string; entryId: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/revisions`;
    return this.client.request<void>('GET', path);
  }

  async getRevision(params: { typeSlug: string; entryId: string; version: number }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/revisions/${params.version}`;
    return this.client.request<void>('GET', path);
  }

  async restoreRevision(params: { entryId: string; version: number; typeSlug: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/revisions/${params.version}/restore`;
    return this.client.request<void>('POST', path);
  }

  async compareRevisions(params: { typeSlug: string; entryId: string }): Promise<void> {
    const path = `/cms/${params.typeSlug}/${params.entryId}/revisions/compare`;
    return this.client.request<void>('GET', path);
  }

}

export function cmsClient(): CmsPlugin {
  return new CmsPlugin();
}
