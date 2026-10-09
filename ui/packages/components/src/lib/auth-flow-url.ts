/** Carry the post-login destination through verification and back to sign-in. */
export function authFlowUrl(
  target: string,
  origin: string,
  search: string,
  email?: string,
): string {
  const url = new URL(target, origin);
  if (email) url.searchParams.set("email", email);
  const redirect = new URLSearchParams(search).get("redirect");
  if (redirect) url.searchParams.set("redirect", redirect);
  return url.href;
}
