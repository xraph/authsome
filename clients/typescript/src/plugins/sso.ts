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

  async registerProvider(request: types.RegisterProviderRequest): Promise<types.ProviderRegisteredResponse> {
    const path = '/sso/provider/register';
    return this.client.request<types.ProviderRegisteredResponse>('POST', path, {
      body: request,
    });
  }

  async sAMLSPMetadata(): Promise<types.MetadataResponse> {
    const path = '/sso/saml2/sp/metadata';
    return this.client.request<types.MetadataResponse>('GET', path);
  }

  async sAMLLogin(params: { providerId: string }, request: types.SAMLLoginRequest): Promise<types.SAMLLoginResponse> {
    const path = `/sso/saml2/login/${params.providerId}`;
    return this.client.request<types.SAMLLoginResponse>('POST', path, {
      body: request,
    });
  }

  async sAMLCallback(params: { providerId: string }): Promise<types.SSOAuthResponse> {
    const path = `/sso/saml2/callback/${params.providerId}`;
    return this.client.request<types.SSOAuthResponse>('POST', path);
  }

  async oIDCLogin(params: { providerId: string }, request: types.OIDCLoginRequest): Promise<types.OIDCLoginResponse> {
    const path = `/sso/oidc/login/${params.providerId}`;
    return this.client.request<types.OIDCLoginResponse>('POST', path, {
      body: request,
    });
  }

  async oIDCCallback(params: { providerId: string }): Promise<types.SSOAuthResponse> {
    const path = `/sso/oidc/callback/${params.providerId}`;
    return this.client.request<types.SSOAuthResponse>('GET', path);
  }

}

export function ssoClient(): SsoPlugin {
  return new SsoPlugin();
}
