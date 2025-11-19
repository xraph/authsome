// Auto-generated AuthSome client

import { ClientPlugin } from './plugin';
import { createErrorFromResponse } from './errors';
import * as types from './types';

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
}

export class AuthsomeClient {
  private baseURL: string;
  private token?: string;
  private apiKey?: string;
  private apiKeyHeader: string;
  private headers: Record<string, string>;
  private plugins: Map<string, ClientPlugin>;

  constructor(config: AuthsomeClientConfig) {
    this.baseURL = config.baseURL;
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

  getPlugin<T extends ClientPlugin>(id: string): T | undefined {
    return this.plugins.get(id) as T | undefined;
  }

  public async request<T>(
    method: string,
    path: string,
    options?: {
      body?: any;
      query?: Record<string, string>;
      auth?: boolean;
    }
  ): Promise<T> {
    const url = new URL(path, this.baseURL);

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
    const path = '/api/auth/signup';
    return this.request<{ user: types.User; session: types.Session }>('POST', path, {
      body: request,
    });
  }

  async signIn(request: { email: string; password: string }): Promise<{ user: types.User; session: types.Session; requiresTwoFactor: boolean }> {
    const path = '/api/auth/signin';
    return this.request<{ user: types.User; session: types.Session; requiresTwoFactor: boolean }>('POST', path, {
      body: request,
    });
  }

  async signOut(): Promise<{ success: boolean }> {
    const path = '/api/auth/signout';
    return this.request<{ success: boolean }>('POST', path, {
      auth: true,
    });
  }

  async getSession(): Promise<{ user: types.User; session: types.Session }> {
    const path = '/api/auth/session';
    return this.request<{ user: types.User; session: types.Session }>('GET', path, {
      auth: true,
    });
  }

  async updateUser(request: { name?: string; email?: string }): Promise<{ user: types.User }> {
    const path = '/api/auth/user/update';
    return this.request<{ user: types.User }>('POST', path, {
      body: request,
      auth: true,
    });
  }

  async listDevices(): Promise<{ devices: types.Device[] }> {
    const path = '/api/auth/devices';
    return this.request<{ devices: types.Device[] }>('GET', path, {
      auth: true,
    });
  }

  async revokeDevice(request: { deviceId: string }): Promise<{ success: boolean }> {
    const path = '/api/auth/devices/revoke';
    return this.request<{ success: boolean }>('POST', path, {
      body: request,
      auth: true,
    });
  }

}
