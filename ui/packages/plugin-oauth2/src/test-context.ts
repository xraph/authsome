import { vi, type Mock } from "vitest";
import type { PluginContext } from "@authsome/ui-core";

export interface FakeContext {
  ctx: PluginContext;
  /** Every call the plugin made, so a test can assert on method and path. */
  request: Mock;
}

/**
 * A PluginContext that records what the plugin asked for.
 *
 * The return type is spelled out rather than inferred, because an inferred
 * vitest Mock cannot be named from outside its own package and tsc refuses
 * to emit a declaration for it.
 */
export function fakeContext(reply: unknown = {}): FakeContext {
  const request = vi.fn().mockResolvedValue(reply);
  return {
    ctx: {
      request: request as PluginContext["request"],
      getState: () => ({ status: "unauthenticated" }) as never,
      getClientConfig: () => null,
      isServerEnabled: () => true,
      getActiveOrgId: () => null,
      on: () => () => {},
    },
    request,
  };
}
