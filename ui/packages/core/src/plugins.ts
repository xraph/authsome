/**
 * Client plugins: the client side of authsome's server plugins.
 *
 * A server plugin is something you opt into with `authsome.WithPlugin(...)`.
 * Its client half is the same: a package you opt into by adding it to the
 * `plugins` list of `AuthManager` (or `<AuthProvider>`), which then gets an
 * authenticated way to call its routes and hears about sign-in, sign-out,
 * refreshes and organization switches. Core knows nothing about any
 * particular plugin.
 *
 * ```ts
 * import { subscription } from "@authsome/plugin-subscription";
 *
 * const auth = new AuthManager({ baseURL, plugins: [subscription()] });
 * const billing = auth.plugin(subscription);
 * ```
 */

import type { AuthState, ClientConfig } from "./types";

/** Moments in the session a plugin can react to. */
export type AuthEvent = "signedIn" | "signedOut" | "sessionRefreshed" | "orgChanged";

/**
 * What a plugin gets from the manager. Kept deliberately small: once plugin
 * packages version on their own, every member here is public API.
 */
export interface PluginContext {
  /**
   * Call the API as the signed-in user: base URL, publishable key and the
   * session's bearer token (or cookie, in cookie mode) are filled in. Query
   * values that are undefined are left out. Rejects with AuthClientError.
   */
  request<T>(
    method: string,
    path: string,
    init?: {
      query?: Record<string, string | number | boolean | undefined>;
      body?: unknown;
    },
  ): Promise<T>;
  /** The current auth state. */
  getState(): AuthState;
  /** The client config, null until it has loaded. */
  getClientConfig(): ClientConfig | null;
  /**
   * Whether the server has this plugin installed, from the client config's
   * supported_plugins. null while that isn't known: no publishable key, or
   * the config hasn't loaded yet.
   */
  isServerEnabled(): boolean | null;
  /** The session's active organization, null when it has none. */
  getActiveOrgId(): string | null;
  /** Listen for a lifecycle event. Returns a function that stops listening. */
  on(event: AuthEvent, listener: () => void): () => void;
}

/** A registered plugin: a name matching the server plugin's, and its init. */
export interface AuthPlugin<Api> {
  /** The server plugin's Name(), e.g. "subscription". */
  readonly name: string;
  /** Builds the plugin's API. Runs once, the first time the plugin is used. */
  init(ctx: PluginContext): Api;
}

/**
 * What definePlugin returns: call it to register the plugin (with options,
 * if it takes any), and pass it to `manager.plugin(...)` to look it up.
 */
export interface PluginDefinition<Api, Options = undefined> {
  (...options: OptionsArgs<Options>): AuthPlugin<Api>;
  readonly pluginName: string;
}

/** Options are optional to pass unless the plugin's init requires them. */
type OptionsArgs<Options> = undefined extends Options
  ? [options?: Options]
  : [options: Options];

/** Either a definition or a registered instance; both identify a plugin. */
export type PluginKey<Api> =
  | AuthPlugin<Api>
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- the lookup only reads pluginName
  | PluginDefinition<Api, any>;

/**
 * Defines a client plugin. `name` must be the server plugin's name, so the
 * plugin can tell from the client config whether the server side is there.
 */
export function definePlugin<Api, Options = undefined>(
  name: string,
  init: (ctx: PluginContext, options: Options) => Api,
): PluginDefinition<Api, Options> {
  const define = (...args: OptionsArgs<Options>): AuthPlugin<Api> => ({
    name,
    init: (ctx) => init(ctx, args[0] as Options),
  });
  return Object.assign(define, { pluginName: name });
}

/** Thrown when code asks for a plugin that wasn't added to the plugins list. */
export class PluginNotRegisteredError extends Error {
  constructor(readonly pluginName: string) {
    super(
      `authsome: the "${pluginName}" plugin isn't registered. ` +
        `Add ${pluginName}() to the plugins of your AuthManager or <AuthProvider>.`,
    );
    this.name = "PluginNotRegisteredError";
  }
}

/** The name a key refers to. */
export function pluginKeyName(key: PluginKey<unknown>): string {
  return "pluginName" in key ? key.pluginName : key.name;
}
