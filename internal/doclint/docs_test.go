package doclint

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// Every pkg.Identifier a security page's Go sample names, for a package of
// this module, exists. A page that shows an API that is not there is worse
// than no page.
func TestSecurityDocsResolve(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	exported, err := ExportedIdentifiers(root)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := ReferencesInDocs(filepath.Join(root, "docs", "content", "docs", "security"))
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) == 0 {
		t.Fatal("no references found; the scanner or the docs moved")
	}
	// Local variable names a sample uses for engine values. A method on
	// them resolves against the Engine type.
	aliases := map[string]string{"engine": "Engine", "eng": "Engine", "e": "Engine", "auth": "Engine"}
	var missing []string
	for _, r := range refs {
		if typ, ok := aliases[r.Package]; ok {
			if !exported["authsome"][typ+"."+r.Name] {
				missing = append(missing, fmt.Sprintf("%s:%d: %s.%s is not a method of Engine", r.File, r.Line, r.Package, r.Name))
			}
			continue
		}
		ex, ok := exported[r.Package]
		if !ok {
			continue // not one of ours (net/http, time, ...)
		}
		if !ex[r.Name] {
			missing = append(missing, fmt.Sprintf("%s:%d: %s.%s does not exist", r.File, r.Line, r.Package, r.Name))
		}
	}
	if len(missing) > 0 {
		t.Errorf("security docs name %d identifiers that do not exist:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}
