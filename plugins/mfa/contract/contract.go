// Package contract wires the mfa plugin's deep-link settings page into
// the forge-dashboard contract registry. Settings flow through
// settings.tabs auto-discovery; no plugin-owned intents yet.
//
// Future iterations may add `mfa.enrollments` (per-user enrolled
// factors) and `mfa.coverage` (deployment-wide MFA adoption stat) when
// the user-detail extensions slot is wired up.
package contract

import (
	"bytes"
	_ "embed"
	"fmt"

	authsome "github.com/xraph/authsome"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/dispatcher"
	"github.com/xraph/forge/extensions/dashboard/contract/loader"
)

//go:embed manifest.yaml
var manifestYAML []byte

type Deps struct {
	Engine *authsome.Engine
}

func Register(
	d *dispatcher.Dispatcher,
	reg contract.Registry,
	wreg contract.WardenRegistry,
	deps Deps,
) error {
	if deps.Engine == nil {
		return fmt.Errorf("mfa/contract: Engine is required")
	}
	m, err := loader.Load(bytes.NewReader(manifestYAML), "mfa/contract/manifest.yaml")
	if err != nil {
		return fmt.Errorf("mfa/contract: load manifest: %w", err)
	}
	if err := loader.Validate(m, wreg); err != nil {
		return fmt.Errorf("mfa/contract: validate manifest: %w", err)
	}
	if err := reg.Register(m); err != nil {
		return fmt.Errorf("mfa/contract: register manifest: %w", err)
	}
	_ = d
	return nil
}
