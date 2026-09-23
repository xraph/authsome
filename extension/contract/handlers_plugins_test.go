package contract

import (
	"context"
	"testing"

	"github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/store/memory"
)

type inventoryPlugin string

func (p inventoryPlugin) Name() string { return string(p) }

func TestPluginsListHandler(t *testing.T) {
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := authsome.NewEngine(
		authsome.WithStore(memory.New()),
		authsome.WithWarden(w),
		authsome.WithPlugin(inventoryPlugin("zeta")),
		authsome.WithPlugin(inventoryPlugin("alpha")),
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := pluginsListHandler(Deps{Engine: engine})(context.Background(), struct{}{}, contract.Principal{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Plugins) != 2 || result.Plugins[0].Name != "alpha" || result.Plugins[1].Name != "zeta" {
		t.Fatalf("unexpected installed plugins: %+v", result.Plugins)
	}
}
