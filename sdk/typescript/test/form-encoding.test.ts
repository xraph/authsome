import { describe, expect, it } from 'vitest';

import { AuthClient } from '../src/index';

/**
 * Builds a client whose transport records the body instead of sending it, so
 * the assertions below read what actually went on the wire rather than what
 * the encoder was asked to produce.
 */
function clientRecordingBody(): { client: AuthClient; body: () => string } {
  let sent = '';

  const client = new AuthClient({
    baseURL: 'https://auth.example.com',
    fetch: async (_url, init) => {
      sent = String(init?.body ?? '');
      return new Response('{"access_token":"at","token_type":"Bearer","expires_in":3600}', {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    },
  });

  return { client, body: () => sent };
}

describe('form-encoded request bodies', () => {
  // RFC 8707 makes `resource` repeatable, and the server reads every value it
  // is given. One value per field is the only shape that survives that, so
  // these assert on getAll() rather than on the joined string.
  it('sends a one-element array as a single field', async () => {
    const { client, body } = clientRecordingBody();

    await client.oauth2Token({
      grant_type: 'client_credentials',
      resource: ['https://api.example.com'],
    });

    expect(new URLSearchParams(body()).getAll('resource')).toEqual(['https://api.example.com']);
  });

  it('repeats the field once per element for a two-element array', async () => {
    const { client, body } = clientRecordingBody();

    await client.oauth2Token({
      grant_type: 'client_credentials',
      resource: ['https://a.example.com', 'https://b.example.com'],
    });

    expect(new URLSearchParams(body()).getAll('resource')).toEqual([
      'https://a.example.com',
      'https://b.example.com',
    ]);
  });

  it('keeps an empty array off the wire entirely', async () => {
    const { client, body } = clientRecordingBody();

    await client.oauth2Token({ grant_type: 'client_credentials', resource: [] });

    expect(new URLSearchParams(body()).has('resource')).toBe(false);
  });

  it('still writes a scalar field as one value', async () => {
    const { client, body } = clientRecordingBody();

    await client.oauth2Token({
      grant_type: 'authorization_code',
      code: 'the-code',
      resource: ['https://api.example.com'],
    });

    expect(new URLSearchParams(body()).get('grant_type')).toBe('authorization_code');
  });

  it('omits an absent optional field', async () => {
    const { client, body } = clientRecordingBody();

    await client.oauth2Token({ grant_type: 'client_credentials' });

    expect(new URLSearchParams(body()).has('resource')).toBe(false);
  });
});

/**
 * Builds a client whose transport records the URL it was called with.
 */
function clientRecordingURL(): { client: AuthClient; url: () => string } {
  let sent = '';

  const client = new AuthClient({
    baseURL: 'https://auth.example.com',
    fetch: async (url) => {
      sent = String(url);
      return new Response(null, { status: 204 });
    },
  });

  return { client, url: () => sent };
}

describe('query-string parameters', () => {
  // A query string carries repeated keys, the same as a form body does. The
  // authorization endpoint reads every `resource` it is given, so an array has
  // to be written one element at a time rather than stringified.
  it('sends a one-element array as a single parameter', async () => {
    const { client, url } = clientRecordingURL();

    await client.oauth2Authorize('code', 'the-client', undefined, undefined, undefined, undefined, undefined, [
      'https://api.example.com',
    ]);

    expect(new URL(url()).searchParams.getAll('resource')).toEqual(['https://api.example.com']);
  });

  it('repeats the parameter once per element for a two-element array', async () => {
    const { client, url } = clientRecordingURL();

    await client.oauth2Authorize('code', 'the-client', undefined, undefined, undefined, undefined, undefined, [
      'https://a.example.com',
      'https://b.example.com',
    ]);

    expect(new URL(url()).searchParams.getAll('resource')).toEqual([
      'https://a.example.com',
      'https://b.example.com',
    ]);
  });

  it('keeps an absent array off the query string', async () => {
    const { client, url } = clientRecordingURL();

    await client.oauth2Authorize('code', 'the-client');

    expect(new URL(url()).searchParams.has('resource')).toBe(false);
  });
});
