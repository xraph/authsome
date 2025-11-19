// Auto-generated sso plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class SsoPlugin implements ClientPlugin {
  readonly id = 'sso';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async registerProvider(request: types.RegisterProvider_req): Promise<types.StatusResponse> {
    const path = '/provider/register';
    return this.client.request<types.StatusResponse>('POST', path, {
      body: request,
    });
  }

  async sAMLSPMetadata(): Promise<types.MetadataResponse> {
    const path = '/saml2/sp/metadata';
    return this.client.request<types.MetadataResponse>('GET', path);
  }

  async sAMLCallback(): Promise<types.StatusResponse> {
    const path = '/saml2/callback/{providerId}';
    return this.client.request<types.StatusResponse>('POST', path);
  }

  async sAMLLogin(): Promise<void> {
    const path = '/saml2/login/{providerId}';
    return this.client.request<void>('GET', path);
  }

  async oIDCCallback(): Promise<types.StatusResponse> {
    const path = '/oidc/callback/{providerId}';
    return this.client.request<types.StatusResponse>('GET', path);
  }

}

export function ssoClient(): SsoPlugin {
  return new SsoPlugin();
}
