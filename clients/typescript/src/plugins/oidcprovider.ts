// Auto-generated oidcprovider plugin

import { ClientPlugin } from '../plugin';
import { AuthsomeClient } from '../client';
import * as types from '../types';

export class OidcproviderPlugin implements ClientPlugin {
  readonly id = 'oidcprovider';
  private client!: AuthsomeClient;

  init(client: AuthsomeClient): void {
    this.client = client;
  }

  async registerClient(request: types.ClientRegistrationRequest): Promise<types.ClientRegistrationResponse> {
    const path = '/oauth2/register';
    return this.client.request<types.ClientRegistrationResponse>('POST', path, {
      body: request,
    });
  }

  async listClients(): Promise<types.ClientsListResponse> {
    const path = '/oauth2/clients';
    return this.client.request<types.ClientsListResponse>('GET', path);
  }

  async getClient(params: { clientId: string }): Promise<types.ClientDetailsResponse> {
    const path = `/oauth2/clients/${params.clientId}`;
    return this.client.request<types.ClientDetailsResponse>('GET', path);
  }

  async updateClient(params: { clientId: string }, request: types.ClientUpdateRequest): Promise<types.ClientDetailsResponse> {
    const path = `/oauth2/clients/${params.clientId}`;
    return this.client.request<types.ClientDetailsResponse>('PUT', path, {
      body: request,
    });
  }

  async deleteClient(params: { clientId: string }): Promise<void> {
    const path = `/oauth2/clients/${params.clientId}`;
    return this.client.request<void>('DELETE', path);
  }

  async discovery(): Promise<types.DiscoveryResponse> {
    const path = '/oauth2/.well-known/openid-configuration';
    return this.client.request<types.DiscoveryResponse>('GET', path);
  }

  async jWKS(): Promise<types.JWKSResponse> {
    const path = '/oauth2/jwks';
    return this.client.request<types.JWKSResponse>('GET', path);
  }

  async authorize(): Promise<void> {
    const path = '/oauth2/authorize';
    return this.client.request<void>('GET', path);
  }

  async handleConsent(request: types.ConsentRequest): Promise<void> {
    const path = '/oauth2/consent';
    return this.client.request<void>('POST', path, {
      body: request,
    });
  }

  async token(request: types.TokenRequest): Promise<types.TokenResponse> {
    const path = '/oauth2/token';
    return this.client.request<types.TokenResponse>('POST', path, {
      body: request,
    });
  }

  async userInfo(): Promise<types.UserInfoResponse> {
    const path = '/oauth2/userinfo';
    return this.client.request<types.UserInfoResponse>('GET', path);
  }

  async introspectToken(request: types.TokenIntrospectionRequest): Promise<types.TokenIntrospectionResponse> {
    const path = '/oauth2/introspect';
    return this.client.request<types.TokenIntrospectionResponse>('POST', path, {
      body: request,
    });
  }

  async revokeToken(request: types.TokenRevocationRequest): Promise<types.StatusResponse> {
    const path = '/oauth2/revoke';
    return this.client.request<types.StatusResponse>('POST', path, {
      body: request,
    });
  }

  async deviceAuthorize(request: types.DeviceAuthorizationRequest): Promise<types.DeviceAuthorizationResponse> {
    const path = '/oauth2/device_authorization';
    return this.client.request<types.DeviceAuthorizationResponse>('POST', path, {
      body: request,
    });
  }

  async deviceCodeEntry(): Promise<types.DeviceCodeEntryResponse> {
    const path = '/oauth2/device';
    return this.client.request<types.DeviceCodeEntryResponse>('GET', path);
  }

  async deviceVerify(request: types.DeviceVerificationRequest): Promise<types.DeviceVerifyResponse> {
    const path = '/oauth2/device/verify';
    return this.client.request<types.DeviceVerifyResponse>('POST', path, {
      body: request,
    });
  }

  async deviceAuthorizeDecision(request: types.DeviceAuthorizationDecisionRequest): Promise<types.DeviceDecisionResponse> {
    const path = '/oauth2/device/authorize';
    return this.client.request<types.DeviceDecisionResponse>('POST', path, {
      body: request,
    });
  }

}

export function oidcproviderClient(): OidcproviderPlugin {
  return new OidcproviderPlugin();
}
