// Auto-generated permissions plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class PermissionsPlugin implements ClientPlugin {
  readonly id = 'permissions';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async migrateAll(request: types.MigrateAllRequest): Promise<types.ErrorResponse> {
    const path = '/permissions/migrate/all';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

  async migrateRoles(): Promise<types.ErrorResponse> {
    const path = '/permissions/migrate/roles';
    return this.client.request<types.ErrorResponse>('POST', path);
  }

  async previewConversion(request: types.PreviewConversionRequest): Promise<types.ErrorResponse> {
    const path = '/permissions/migrate/preview';
    return this.client.request<types.ErrorResponse>('POST', path, {
      body: request,
    });
  }

}

export function permissionsClient(): PermissionsPlugin {
  return new PermissionsPlugin();
}
