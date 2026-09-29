package doclint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Exported is what a package exports, by identifier: top-level names and
// the methods of its types. Methods are keyed as Type.Method as well as
// Method alone, so a sample calling engine.SignIn resolves by method name.
type Exported map[string]bool

// ExportedIdentifiers parses every non-test Go file under root (skipping
// vendor, testdata and generated SDKs) and returns exports by package name.
// Two packages sharing a name are merged, which is the right answer for a
// doc sample that names only the short package name.
func ExportedIdentifiers(root string) (map[string]Exported, error) {
	out := make(map[string]Exported)
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "vendor" || name == "testdata" || name == "node_modules" || name == "sdk" || name == "docs" || name == "flutter" || name == "ui" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil // a file that does not parse is the compiler's problem, not the docs'
		}
		pkg := f.Name.Name
		ex, ok := out[pkg]
		if !ok {
			ex = make(Exported)
			out[pkg] = ex
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if !d.Name.IsExported() {
					continue
				}
				ex[d.Name.Name] = true
				if d.Recv != nil && len(d.Recv.List) > 0 {
					if tn := receiverTypeName(d.Recv.List[0].Type); tn != "" {
						ex[tn+"."+d.Name.Name] = true
					}
				}
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							ex[s.Name.Name] = true
							if st, ok := s.Type.(*ast.StructType); ok {
								for _, fld := range st.Fields.List {
									for _, n := range fld.Names {
										if n.IsExported() {
											ex[s.Name.Name+"."+n.Name] = true
											ex[n.Name] = true
										}
									}
								}
							}
						}
					case *ast.ValueSpec:
						for _, n := range s.Names {
							if n.IsExported() {
								ex[n.Name] = true
							}
						}
					}
				}
			}
		}
		return nil
	})
	return out, err
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return ""
}

// Reference is one pkg.Identifier a doc sample names.
type Reference struct {
	File    string
	Line    int
	Package string
	Name    string
}

var (
	goFence  = regexp.MustCompile("(?s)```go\\n(.*?)```")
	selector = regexp.MustCompile(`\b([a-z][a-z0-9]*)\.([A-Z][A-Za-z0-9]*)`)
)

// ReferencesInDocs extracts every pkg.Identifier selector from the Go
// fences of the .mdx files under dir, with the line it sits on.
func ReferencesInDocs(dir string) ([]Reference, error) {
	var refs []Reference
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".mdx") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		src, readErr := os.ReadFile(path) // #nosec G304 -- path comes from listing the docs directory, not from input
		if readErr != nil {
			return nil, readErr
		}
		text := string(src)
		for _, m := range goFence.FindAllStringSubmatchIndex(text, -1) {
			body := text[m[2]:m[3]]
			baseLine := strings.Count(text[:m[2]], "\n") + 1
			for i, line := range strings.Split(body, "\n") {
				code := line
				if idx := strings.Index(code, "//"); idx >= 0 {
					code = code[:idx]
				}
				for _, sm := range selector.FindAllStringSubmatch(code, -1) {
					refs = append(refs, Reference{File: e.Name(), Line: baseLine + i, Package: sm[1], Name: sm[2]})
				}
			}
		}
	}
	return refs, nil
}
