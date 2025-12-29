// Auto-generated AuthSome client

import { ClientPlugin } from './plugin';
import { createErrorFromResponse } from './errors';
import * as types from './types';
import { AdminPlugin } from './plugins/admin';
import { PermissionsPlugin } from './plugins/permissions';
import { WebhookPlugin } from './plugins/webhook';
import { CmsPlugin } from './plugins/cms';
import { EmailverificationPlugin } from './plugins/emailverification';
import { IdverificationPlugin } from './plugins/idverification';
import { MagiclinkPlugin } from './plugins/magiclink';
import { NotificationPlugin } from './plugins/notification';
import { TwofaPlugin } from './plugins/twofa';
import { ApikeyPlugin } from './plugins/apikey';
import { EmailotpPlugin } from './plugins/emailotp';
import { BackupauthPlugin } from './plugins/backupauth';
import { CompliancePlugin } from './plugins/compliance';
import { StepupPlugin } from './plugins/stepup';
import { MfaPlugin } from './plugins/mfa';
import { MultiappPlugin } from './plugins/multiapp';
import { PhonePlugin } from './plugins/phone';
import { AnonymousPlugin } from './plugins/anonymous';
import { PasskeyPlugin } from './plugins/passkey';
import { UsernamePlugin } from './plugins/username';
import { MultisessionPlugin } from './plugins/multisession';
import { ConsentPlugin } from './plugins/consent';
import { ImpersonationPlugin } from './plugins/impersonation';
import { OidcproviderPlugin } from './plugins/oidcprovider';
import { OrganizationPlugin } from './plugins/organization';
import { SsoPlugin } from './plugins/sso';
import { SecretsPlugin } from './plugins/secrets';
import { SocialPlugin } from './plugins/social';
import { JwtPlugin } from './plugins/jwt';

/**
 * AuthSome client configuration
 * Supports multiple authentication methods that can be used simultaneously:
 * - Cookies: Automatically sent with every request (session-based auth)
 * - Bearer Token: JWT tokens sent in Authorization header when auth: true
 * - API Key: Sent with every request for server-to-server auth
 *   - Publishable Key (pk_*): Safe for frontend, limited permissions
 *   - Secret Key (sk_*): Backend only, full admin access
 */
export interface AuthsomeClientConfig {
  /** Base URL of the AuthSome API */
  baseURL: string;
  
  /** Plugin instances to initialize */
  plugins?: ClientPlugin[];
  
  /** JWT/Bearer token for user authentication (sent only when auth: true) */
  token?: string;
  
  /** API key for server-to-server auth (pk_* or sk_*, sent with all requests) */
  apiKey?: string;
  
  /** Custom header name for API key (default: 'X-API-Key') */
  apiKeyHeader?: string;
  
  /** Custom headers to include with all requests */
  headers?: Record<string, string>;
  
  /** Base path prefix for all API routes (default: '') */
  basePath?: string;
}

export class AuthsomeClient {
  private baseURL: string;
  private basePath: string;
  private token?: string;
  private apiKey?: string;
  private apiKeyHeader: string;
  private headers: Record<string, string>;
  private plugins: Map<string, ClientPlugin>;

  constructor(config: AuthsomeClientConfig) {
    this.baseURL = config.baseURL;
    this.basePath = config.basePath || '';
    this.token = config.token;
    this.apiKey = config.apiKey;
    this.apiKeyHeader = config.apiKeyHeader || 'X-API-Key';
    this.headers = config.headers || {};
    this.plugins = new Map();

    if (config.plugins) {
      for (const plugin of config.plugins) {
        this.plugins.set(plugin.id, plugin);
        plugin.init(this);
      }
    }
  }

  setToken(token: string): void {
    this.token = token;
  }

  setApiKey(apiKey: string, header?: string): void {
    this.apiKey = apiKey;
    if (header) {
      this.apiKeyHeader = header;
    }
  }

  /**
   * Set a publishable key (pk_*) - safe for frontend use
   * Publishable keys have limited permissions and can be exposed in client-side code
   * Typically used for: session creation, user verification, public data reads
   */
  setPublishableKey(publishableKey: string): void {
    if (!publishableKey.startsWith('pk_')) {
      console.warn('Warning: Publishable keys should start with pk_');
    }
    this.setApiKey(publishableKey);
  }

  /**
   * Set a secret key (sk_*) - MUST be kept secret on server-side only!
   * Secret keys have full administrative access to all operations
   * WARNING: Never expose secret keys in client-side code (browser, mobile apps)
   */
  setSecretKey(secretKey: string): void {
    if (!secretKey.startsWith('sk_')) {
      console.warn('Warning: Secret keys should start with sk_');
    }
    this.setApiKey(secretKey);
  }

  setBasePath(basePath: string): void {
    this.basePath = basePath;
  }

  /**
   * Convert an object to query parameters, handling optional values and type conversion
   */
  public toQueryParams(obj?: Record<string, any>): Record<string, string> | undefined {
    if (!obj) return undefined;
    
    const params: Record<string, string> = {};
    for (const [key, value] of Object.entries(obj)) {
      if (value !== undefined && value !== null) {
        params[key] = String(value);
      }
    }
    return Object.keys(params).length > 0 ? params : undefined;
  }

  /**
   * Set global headers for all requests
   * @param headers - Headers to set
   * @param replace - If true, replaces all existing headers. If false (default), merges with existing headers
   */
  setGlobalHeaders(headers: Record<string, string>, replace: boolean = false): void {
    if (replace) {
      this.headers = { ...headers };
    } else {
      this.headers = { ...this.headers, ...headers };
    }
  }

  getPlugin<T extends ClientPlugin>(id: string): T | undefined {
    return this.plugins.get(id) as T | undefined;
  }

  public readonly $plugins = {
    admin: (): AdminPlugin | undefined => this.getPlugin<AdminPlugin>('admin'),
    permissions: (): PermissionsPlugin | undefined => this.getPlugin<PermissionsPlugin>('permissions'),
    webhook: (): WebhookPlugin | undefined => this.getPlugin<WebhookPlugin>('webhook'),
    cms: (): CmsPlugin | undefined => this.getPlugin<CmsPlugin>('cms'),
    emailverification: (): EmailverificationPlugin | undefined => this.getPlugin<EmailverificationPlugin>('emailverification'),
    idverification: (): IdverificationPlugin | undefined => this.getPlugin<IdverificationPlugin>('idverification'),
    magiclink: (): MagiclinkPlugin | undefined => this.getPlugin<MagiclinkPlugin>('magiclink'),
    notification: (): NotificationPlugin | undefined => this.getPlugin<NotificationPlugin>('notification'),
    twofa: (): TwofaPlugin | undefined => this.getPlugin<TwofaPlugin>('twofa'),
    apikey: (): ApikeyPlugin | undefined => this.getPlugin<ApikeyPlugin>('apikey'),
    emailotp: (): EmailotpPlugin | undefined => this.getPlugin<EmailotpPlugin>('emailotp'),
    backupauth: (): BackupauthPlugin | undefined => this.getPlugin<BackupauthPlugin>('backupauth'),
    compliance: (): CompliancePlugin | undefined => this.getPlugin<CompliancePlugin>('compliance'),
    stepup: (): StepupPlugin | undefined => this.getPlugin<StepupPlugin>('stepup'),
    mfa: (): MfaPlugin | undefined => this.getPlugin<MfaPlugin>('mfa'),
    multiapp: (): MultiappPlugin | undefined => this.getPlugin<MultiappPlugin>('multiapp'),
    phone: (): PhonePlugin | undefined => this.getPlugin<PhonePlugin>('phone'),
    anonymous: (): AnonymousPlugin | undefined => this.getPlugin<AnonymousPlugin>('anonymous'),
    passkey: (): PasskeyPlugin | undefined => this.getPlugin<PasskeyPlugin>('passkey'),
    username: (): UsernamePlugin | undefined => this.getPlugin<UsernamePlugin>('username'),
    multisession: (): MultisessionPlugin | undefined => this.getPlugin<MultisessionPlugin>('multisession'),
    consent: (): ConsentPlugin | undefined => this.getPlugin<ConsentPlugin>('consent'),
    impersonation: (): ImpersonationPlugin | undefined => this.getPlugin<ImpersonationPlugin>('impersonation'),
    oidcprovider: (): OidcproviderPlugin | undefined => this.getPlugin<OidcproviderPlugin>('oidcprovider'),
    organization: (): OrganizationPlugin | undefined => this.getPlugin<OrganizationPlugin>('organization'),
    sso: (): SsoPlugin | undefined => this.getPlugin<SsoPlugin>('sso'),
    secrets: (): SecretsPlugin | undefined => this.getPlugin<SecretsPlugin>('secrets'),
    social: (): SocialPlugin | undefined => this.getPlugin<SocialPlugin>('social'),
    jwt: (): JwtPlugin | undefined => this.getPlugin<JwtPlugin>('jwt'),
  };

  public async request<T>(
    method: string,
    path: string,
    options?: {
      body?: any;
      query?: Record<string, string>;
      auth?: boolean;
    }
  ): Promise<T> {
    const url = new URL(this.basePath + path, this.baseURL);

    if (options?.query) {
      for (const [key, value] of Object.entries(options.query)) {
        url.searchParams.append(key, value);
      }
    }

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...this.headers,
    };

    if (options?.auth && this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    if (this.apiKey) {
      headers[this.apiKeyHeader] = this.apiKey;
    }

    const response = await fetch(url.toString(), {
      method,
      headers,
      body: options?.body ? JSON.stringify(options.body) : undefined,
      credentials: 'include',
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: response.statusText }));
      throw createErrorFromResponse(response.status, error.error || error.message || 'Request failed');
    }

    return response.json();
  }

  async signUp(request: { email: string; password: string; name?: string }): Promise<{ user: types.User; session: types.Session }> {
    const path = '/signup';
    return this.request<{ user: types.User; session: types.Session }>('POST', path, {
      body: request,
    });
  }

  async signIn(request: { email: string; password: string }): Promise<{ user: types.User; session: types.Session; requiresTwoFactor: boolean }> {
    const path = '/signin';
    return this.request<{ user: types.User; session: types.Session; requiresTwoFactor: boolean }>('POST', path, {
      body: request,
    });
  }

  async signOut(): Promise<{ success: boolean }> {
    const path = '/signout';
    return this.request<{ success: boolean }>('POST', path, {
      auth: true,
    });
  }

  async getSession(): Promise<{ user: types.User; session: types.Session }> {
    const path = '/session';
    return this.request<{ user: types.User; session: types.Session }>('GET', path, {
      auth: true,
    });
  }

  async updateUser(request: { email?: string; name?: string }): Promise<{ user: types.User }> {
    const path = '/user/update';
    return this.request<{ user: types.User }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async listDevices(): Promise<{ devices: types.Device[] }> {
    const path = '/devices';
    return this.request<{ devices: types.Device[] }>('GET', path, {
      auth: true,
    });
  }

  async revokeDevice(request: { fingerprint: string }): Promise<{ status: string }> {
    const path = '/devices/revoke';
    return this.request<{ status: string }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async refreshSession(request: { refreshToken: string }): Promise<{ expiresAt: string; refreshExpiresAt: string; session: any; accessToken: string; refreshToken: string }> {
    const path = '/refresh';
    return this.request<{ session: any; accessToken: string; refreshToken: string; expiresAt: string; refreshExpiresAt: string }>('POST', path, {
      body: request,
    });
  }

  async requestPasswordReset(request: { email: string }): Promise<{ message: string }> {
    const path = '/password/reset/request';
    return this.request<{ message: string }>('POST', path, {
      body: request,
    });
  }

  async resetPassword(request: { token: string; newPassword: string }): Promise<{ message: string }> {
    const path = '/password/reset/confirm';
    return this.request<{ message: string }>('POST', path, {
      body: request,
    });
  }

  async validateResetToken(query?: { token?: string }): Promise<{ valid: boolean }> {
    const path = '/password/reset/validate';
    return this.request<{ valid: boolean }>('GET', path, {
      query,
    });
  }

  async changePassword(request: { oldPassword: string; newPassword: string }): Promise<{ message: string }> {
    const path = '/password/change';
    return this.request<{ message: string }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async requestEmailChange(request: { newEmail: string }): Promise<{ message: string }> {
    const path = '/email/change/request';
    return this.request<{ message: string }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async confirmEmailChange(request: { token: string }): Promise<{ message: string }> {
    const path = '/email/change/confirm';
    return this.request<{ message: string }>('POST', path, {
      body: request,
    });
  }

}
