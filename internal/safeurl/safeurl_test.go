package safeurl

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResolver answers from a table; an absent name does not resolve.
type fakeResolver map[string][]string

func (r fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := r[host]
	if !ok {
		return nil, &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	}
	out := make([]net.IPAddr, len(ips))
	for i, ip := range ips {
		out[i] = net.IPAddr{IP: net.ParseIP(ip)}
	}
	return out, nil
}

func TestValidate(t *testing.T) {
	opts := Options{Resolver: fakeResolver{
		"public.example":    {"93.184.216.34"},
		"private.example":   {"10.1.2.3"},
		"mixed.example":     {"93.184.216.34", "10.1.2.3"},
		"metadata.example":  {"169.254.169.254"},
		"mapped.example":    {"::ffff:127.0.0.1"},
		"v6public.example":  {"2606:2800:220:1:248:1893:25c8:1946"},
		"v6private.example": {"fd12::1"},
	}}
	ctx := context.Background()
	cases := []struct {
		name, raw string
		want      error
	}{
		{"public name", "https://public.example/hook", nil},
		{"public v6", "https://v6public.example/hook", nil},
		{"http refused", "http://public.example/hook", ErrScheme},
		{"other scheme", "ftp://public.example/", ErrScheme},
		{"no host", "https:///hook", ErrHost},
		{"credentials", "https://user:pw@public.example/hook", ErrUserinfo},
		{"private name", "https://private.example/hook", ErrNonPublic},
		{"one private answer poisons the name", "https://mixed.example/hook", ErrNonPublic},
		{"cloud metadata", "https://metadata.example/hook", ErrNonPublic},
		{"mapped loopback", "https://mapped.example/hook", ErrNonPublic},
		{"private v6", "https://v6private.example/hook", ErrNonPublic},
		{"literal loopback", "https://127.0.0.1/hook", ErrNonPublic},
		{"literal metadata", "https://169.254.169.254/latest", ErrNonPublic},
		{"literal v6 loopback", "https://[::1]/hook", ErrNonPublic},
		{"literal cgnat", "https://100.64.0.1/hook", ErrNonPublic},
		{"unresolvable", "https://nowhere.example/hook", ErrResolve},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Validate(ctx, tc.raw, opts)
			if tc.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tc.want)
		})
	}

	u, err := Validate(ctx, "http://127.0.0.1:8080/hook", Options{AllowInsecure: true})
	require.NoError(t, err, "insecure mode is for development against a local receiver")
	assert.Equal(t, "127.0.0.1:8080", u.Host)
}

func TestIsPublic(t *testing.T) {
	for _, s := range []string{"0.0.0.0", "10.0.0.1", "172.16.0.1", "192.168.1.1", "127.0.0.1", "169.254.169.254",
		"100.64.0.1", "192.0.0.1", "198.18.0.1", "240.0.0.1", "224.0.0.1", "::1", "::", "fe80::1", "fc00::1",
		"64:ff9b::7f00:1", "2001:db8::1", "::ffff:10.0.0.1"} {
		assert.False(t, IsPublic(netip.MustParseAddr(s)), s)
	}
	for _, s := range []string{"93.184.216.34", "8.8.8.8", "2606:4700::1111"} {
		assert.True(t, IsPublic(netip.MustParseAddr(s)), s)
	}
}

func TestClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/ok", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	ctx := context.Background()

	insecure := Client(time.Second, Options{AllowInsecure: true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/ok", http.NoBody)
	require.NoError(t, err)
	resp, err := insecure.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/redirect", http.NoBody)
	require.NoError(t, err)
	resp, err = insecure.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusFound, resp.StatusCode, "a redirect is handed back, never followed")

	strict := Client(time.Second, Options{})
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/ok", http.NoBody)
	require.NoError(t, err)
	_, err = strict.Do(req) //nolint:bodyclose // the request never connects
	assert.ErrorIs(t, err, ErrNonPublic, "the dial refuses the loopback address the test server lives on")
}
