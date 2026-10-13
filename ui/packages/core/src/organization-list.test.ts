import { describe, expect, it } from "vitest";

import { readOrganizationList } from "./client";

const acme = { id: "org_1", name: "Acme" } as never;

describe("readOrganizationList", () => {
  it("reads the paged { organizations, next_cursor } shape", () => {
    expect(readOrganizationList({ organizations: [acme], next_cursor: "cur_2" })).toEqual({
      organizations: [acme],
      nextCursor: "cur_2",
      total: 1,
    });
  });

  it("reads the older { items, total } shape", () => {
    expect(readOrganizationList({ items: [acme], total: 9 })).toEqual({
      organizations: [acme],
      nextCursor: undefined,
      total: 9,
    });
  });

  it("treats a missing or null list as empty", () => {
    expect(readOrganizationList({ organizations: null, next_cursor: "" })).toEqual({
      organizations: [],
      nextCursor: undefined,
      total: 0,
    });
    expect(readOrganizationList(undefined).organizations).toEqual([]);
  });
});
