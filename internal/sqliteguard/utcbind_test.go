package sqliteguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// timestampComparison matches a WHERE fragment that compares a timestamp
// column against a bound parameter: `expires_at > ?`, `received_at >= ?`,
// `deleted_at < ?`. It deliberately does not match equality, which is not
// affected by ordering, or a Set clause, which writes rather than compares.
var timestampComparison = regexp.MustCompile(`\b\w*(_at|expires\w*)\s*[<>]=?\s*\?`)

// TestTimestampComparisonsBindUTC is the guard on a bug this repo has now hit
// five times.
//
// SQLite has no timestamp type. These schemas store timestamps as TEXT, so a
// predicate like `expires_at > ?` is a string comparison, and it is only
// correct while both sides carry the same wall clock. Bind a local-zone
// time.Time against a stored UTC value and the answer is wrong by the zone
// offset, in a direction that depends on which side of UTC the process runs.
// West of UTC an expiry check keeps expired rows alive. East of it a rate
// limit stops counting.
//
// None of those failures look like a timezone bug from the outside, they look
// like a token that would not die or a limiter that would not trip, so this
// asserts the invariant rather than trusting anybody to remember it: every
// timestamp comparison in a SQLite store binds a value that has been put on
// UTC, either with .UTC() or through the local utc() helper.
//
// A Go-side comparison is not affected and is not checked. time.Time compares
// instants, not text, so two correctly constructed times compare correctly
// whatever zone they carry. The bug only exists where the comparison happens
// inside the database.
func TestTimestampComparisonsBindUTC(t *testing.T) {
	root := repoRoot(t)

	files := []string{}
	sqliteDir := filepath.Join(root, "store", "sqlite")
	entries, err := os.ReadDir(sqliteDir)
	require.NoError(t, err, "read store/sqlite")
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, filepath.Join(sqliteDir, e.Name()))
		}
	}
	plugins, err := filepath.Glob(filepath.Join(root, "plugins", "*", "store_sqlite.go"))
	require.NoError(t, err)
	files = append(files, plugins...)

	require.NotEmpty(t, files, "found no sqlite store files to check; has the layout moved?")

	var violations []string
	fset := token.NewFileSet()
	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		require.NoError(t, err, "parse %s", path)

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Where" {
				return true
			}
			lit, ok := call.Args[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			clause, err := strconv.Unquote(lit.Value)
			if err != nil || !timestampComparison.MatchString(clause) {
				return true
			}
			if isUTCNormalized(call.Args[1]) {
				return true
			}
			pos := fset.Position(call.Pos())
			violations = append(violations, "  "+relTo(root, pos.Filename)+":"+
				strconv.Itoa(pos.Line)+"  Where("+strconv.Quote(clause)+", "+
				exprString(call.Args[1])+")")
			return true
		})
	}

	if len(violations) > 0 {
		t.Fatalf("timestamp comparisons in SQLite stores must bind a UTC value, "+
			"because these columns are TEXT and the comparison is a string sort.\n"+
			"Wrap the bound value in .UTC(), or utc() where the package has one:\n%s",
			strings.Join(violations, "\n"))
	}
}

// isUTCNormalized reports whether an expression has been put on UTC: either
// something.UTC() or a call to the package-local utc() helper.
func isUTCNormalized(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		return fn.Sel.Name == "UTC"
	case *ast.Ident:
		return fn.Name == "utc"
	}
	return false
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return exprString(v.Fun) + "(...)"
	}
	return "the bound value"
}

func relTo(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return rel
	}
	return path
}

// repoRoot walks up from the test's own directory to the module root.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for range 10 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "walked past the filesystem root without finding go.mod")
		dir = parent
	}
	t.Fatal("could not locate the module root")
	return ""
}
