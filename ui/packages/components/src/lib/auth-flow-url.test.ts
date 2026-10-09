import { describe, expect, it } from "vitest";
import { authFlowUrl } from "./auth-flow-url";

describe("verification navigation", () => {
  it("carries the email and destination into verification without dropping existing parameters", () => {
    const result = new URL(
      authFlowUrl(
        "/sign-up/verify-email?locale=en",
        "https://app.test",
        "?redirect=%2Fworkspace%3Ftab%3Dactive",
        "ada+team@example.com",
      ),
    );
    expect(result.pathname).toBe("/sign-up/verify-email");
    expect(result.searchParams.get("email")).toBe("ada+team@example.com");
    expect(result.searchParams.get("redirect")).toBe("/workspace?tab=active");
    expect(result.searchParams.get("locale")).toBe("en");
  });
  it("returns to sign-in with the destination and removes the verification email", () => {
    expect(
      authFlowUrl(
        "/sign-in",
        "https://app.test",
        "?email=ada%40example.com&redirect=%2Fworkspace",
      ),
    ).toBe("https://app.test/sign-in?redirect=%2Fworkspace");
  });
});
