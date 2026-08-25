package sso

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// MemoryStore locks every method, and no test in this package ever called it
// from a second goroutine, so `go test -race` here watched nothing.
//
// Unlike the other stores in this change, nothing here mutates a stored
// *Connection in place: UpdateConnection replaces the map entry. The exposure
// was the other direction. Five separate read methods returned the stored
// pointer, so any caller could write straight through it into the store, and
// that write is unsynchronized against every concurrent reader. What it could
// rewrite is not incidental: Active, Domain, IDPCertificate and
// AttributeMappings between them decide whether a connection is live, which
// domain routes to it, which IdP signature is trusted, and which IdP claim
// becomes which local identity field.

const (
	ssoHammerWorkers = 4
	ssoHammerBudget  = 150 * time.Millisecond
)

func newConnection(appID id.AppID, domain, provider string) *Connection {
	return &Connection{
		ID: id.NewSSOConnectionID(), AppID: appID,
		Provider: provider, Protocol: "saml", Domain: domain,
		Active: true, IDPCertificate: "real-cert",
		AttributeMappings: map[string]string{"email": "mail"},
		CreatedAt:         time.Now(),
	}
}

// TestMemoryStore_ConcurrentReadsAndWrites drives every lookup path against
// create and update, with readers writing to what they are handed.
func TestMemoryStore_ConcurrentReadsAndWrites(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	conns := make([]*Connection, 6)
	for i := range conns {
		conns[i] = newConnection(appID, fmt.Sprintf("d%d.example.com", i), fmt.Sprintf("okta-%d", i))
		require.NoError(t, s.CreateConnection(ctx, conns[i]))
	}

	deadline := time.Now().Add(ssoHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < ssoHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				c := conns[(w+i)%len(conns)]

				if got, err := s.GetConnection(ctx, c.ID); err == nil {
					got.Active = false
					got.IDPCertificate = "forged"
					got.AttributeMappings["email"] = "attacker"
				}
				if got, err := s.GetConnectionByDomain(ctx, appID, c.Domain); err == nil {
					got.Active = false
				}
				if got, err := s.GetConnectionByProvider(ctx, appID, c.Provider); err == nil {
					got.Domain = "evil.example.com"
				}
				if _, err := s.ListConnections(ctx, appID); err != nil {
					t.Errorf("ListConnections: %v", err)
					return
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(500),
		"the hammer did no meaningful work; it is not exercising the store concurrently")

	for i, c := range conns {
		got, err := s.GetConnection(ctx, c.ID)
		require.NoError(t, err)
		require.True(t, got.Active, "connection %d was deactivated by a reader's write", i)
		require.Equal(t, "real-cert", got.IDPCertificate, "connection %d's IdP certificate was rewritten by a reader", i)
		require.Equal(t, "mail", got.AttributeMappings["email"], "connection %d's attribute mapping was rewritten by a reader", i)
		require.Equal(t, c.Domain, got.Domain, "connection %d's domain was rewritten by a reader", i)
	}
}

// TestMemoryStore_GetDoesNotAliasStoredConnection pins the read side, and in
// particular AttributeMappings. That field is a map, so a shallow struct copy
// passes every scalar check here and still leaves the mapping shared.
func TestMemoryStore_GetDoesNotAliasStoredConnection(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	c := newConnection(appID, "acme.example.com", "okta")
	require.NoError(t, s.CreateConnection(ctx, c))

	got, err := s.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	got.Active = false
	got.IDPCertificate = "forged"
	got.AttributeMappings["email"] = "attacker"

	fresh, err := s.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	require.True(t, fresh.Active, "deactivating a returned copy must not deactivate the stored connection")
	require.Equal(t, "real-cert", fresh.IDPCertificate)
	require.Equal(t, "mail", fresh.AttributeMappings["email"],
		"AttributeMappings is a map; a shallow copy would still alias it")
}

// TestMemoryStore_CreateDoesNotAliasCallerConnection is the mirror image.
func TestMemoryStore_CreateDoesNotAliasCallerConnection(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	c := newConnection(appID, "acme.example.com", "okta")
	require.NoError(t, s.CreateConnection(ctx, c))

	c.Active = false
	c.IDPCertificate = "forged"
	c.AttributeMappings["email"] = "attacker"

	got, err := s.GetConnection(ctx, c.ID)
	require.NoError(t, err)
	require.True(t, got.Active, "mutating the caller's connection after Create must not affect the store")
	require.Equal(t, "real-cert", got.IDPCertificate)
	require.Equal(t, "mail", got.AttributeMappings["email"])
}

// TestMemoryStore_ListDoesNotAliasStoredConnections covers the many-pointer
// path, the easiest one to miss when adding a copy.
func TestMemoryStore_ListDoesNotAliasStoredConnections(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()
	for i := 0; i < 3; i++ {
		require.NoError(t, s.CreateConnection(ctx, newConnection(appID, fmt.Sprintf("d%d.example.com", i), fmt.Sprintf("okta-%d", i))))
	}

	first, err := s.ListConnections(ctx, appID)
	require.NoError(t, err)
	require.Len(t, first, 3)
	for _, c := range first {
		c.Active = false
		c.AttributeMappings["email"] = "attacker"
	}

	second, err := s.ListConnections(ctx, appID)
	require.NoError(t, err)
	for _, c := range second {
		require.True(t, c.Active, "mutating a listed connection must not affect the store")
		require.Equal(t, "mail", c.AttributeMappings["email"])
	}
}

// TestMemoryStore_ConcurrentCreateUpdateDelete exercises the write paths
// against each other and against the lookups that scan the whole map.
func TestMemoryStore_ConcurrentCreateUpdateDelete(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	appID := id.NewAppID()

	deadline := time.Now().Add(ssoHammerBudget)
	var wg sync.WaitGroup
	var ops atomic.Int64

	for w := 0; w < ssoHammerWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				c := newConnection(appID, fmt.Sprintf("w%d-%d.example.com", w, i%8), fmt.Sprintf("p%d", w))
				if err := s.CreateConnection(ctx, c); err != nil {
					t.Errorf("CreateConnection: %v", err)
					return
				}
				c.Active = false
				if err := s.UpdateConnection(ctx, c); err != nil {
					t.Errorf("UpdateConnection: %v", err)
					return
				}
				_, _ = s.GetConnectionByDomain(ctx, appID, c.Domain)
				if i%4 == 0 {
					_ = s.DeleteConnection(ctx, c.ID)
				}
				ops.Add(1)
			}
		}(w)
	}
	wg.Wait()

	require.Greater(t, ops.Load(), int64(300), "the hammer did no meaningful work")
}
