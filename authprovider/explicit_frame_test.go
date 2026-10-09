package authprovider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xraph/forge/extensions/auth"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/authprovider"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/internal/dpoptest"
	authmw "github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

func TestSessionProviderExplicitFrameProvenance(t *testing.T) {
	provider := newBoundProvider(t, "")
	req := providerRequest(t, "Bearer", "")
	ctx := authmw.WithCookieBridgedToken(req.Context(), dpopProviderToken)
	bridged, err := provider.Authenticate(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "cookie", bridged.Metadata["credential_scheme"])

	ctx, err = auth.WithExplicitFrameCredential(ctx, "Bearer", dpopProviderToken)
	require.NoError(t, err)
	explicit, err := provider.Authenticate(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "bearer", explicit.Metadata["credential_scheme"])
	require.Equal(t, bridged.Subject, explicit.Subject)
}

func TestSessionProviderExplicitFrameStillValidatesToken(t *testing.T) {
	for _, reason := range []string{"invalid token", "expired token"} {
		t.Run(reason, func(t *testing.T) {
			called := false
			provider := authprovider.NewSessionProvider(
				func(context.Context, string) (*session.Session, error) {
					called = true
					return nil, errors.New(reason)
				},
				func(context.Context, string) (*user.User, error) {
					t.Fatal("invalid session must not resolve a user")
					return nil, errors.New("unexpected user lookup")
				}, log.NewNoopLogger())
			req := providerRequest(t, "Bearer", "")
			ctx, err := auth.WithExplicitFrameCredential(req.Context(), "Bearer", dpopProviderToken)
			require.NoError(t, err)
			result, err := provider.Authenticate(ctx, req)
			require.True(t, called)
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
			require.Nil(t, result)
		})
	}
}

func TestSessionProviderExplicitFrameDPoPNegatives(t *testing.T) {
	bound := dpoptest.Key(t)
	wrong := dpoptest.Key(t)
	for _, tt := range []struct {
		name, scheme, proof string
		validator           bool
	}{
		{"missing proof", "DPoP", "", true},
		{"wrong key", "DPoP", providerProof(t, wrong), true},
		{"bound bearer", "Bearer", providerProof(t, bound), true},
		{"no validator", "DPoP", providerProof(t, bound), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			bind := authmw.SessionBindingConfig{}
			if tt.validator {
				bind.DPoPValidator = dpop.NewValidator(dpop.Config{})
			}
			provider := newBoundProvider(t, dpoptest.Thumbprint(t, bound), bind)
			req := providerRequest(t, tt.scheme, tt.proof)
			ctx := authmw.WithCookieBridgedToken(req.Context(), dpopProviderToken)
			ctx, err := auth.WithExplicitFrameCredential(ctx, tt.scheme, dpopProviderToken)
			require.NoError(t, err)
			result, err := provider.Authenticate(ctx, req)
			require.ErrorIs(t, err, auth.ErrInvalidCredentials)
			require.Nil(t, result)
		})
	}
}

func TestSessionProviderExplicitFramePreservesDPoPScope(t *testing.T) {
	key := dpoptest.Key(t)
	provider := newBoundProvider(t, dpoptest.Thumbprint(t, key), authmw.SessionBindingConfig{
		DPoPValidator: dpop.NewValidator(dpop.Config{}),
	})
	req := providerRequest(t, "DPoP", providerProof(t, key))
	ctx := dpop.WithRequestScope(req.Context())
	ctx = authmw.WithCookieBridgedToken(ctx, dpopProviderToken)
	_, err := provider.Authenticate(ctx, req)
	require.NoError(t, err)

	marked, err := auth.WithExplicitFrameCredential(ctx, "DPoP", dpopProviderToken)
	require.NoError(t, err)
	result, err := provider.Authenticate(marked, req.WithContext(marked))
	require.NoError(t, err)
	require.Equal(t, "dpop", result.Metadata["credential_scheme"])

	fresh := providerRequest(t, "DPoP", req.Header.Get("DPoP"))
	freshCtx, err := auth.WithExplicitFrameCredential(dpop.WithRequestScope(fresh.Context()), "DPoP", dpopProviderToken)
	require.NoError(t, err)
	result, err = provider.Authenticate(freshCtx, fresh.WithContext(freshCtx))
	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	require.Nil(t, result)
}
