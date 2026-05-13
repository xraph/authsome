package wardenseed

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/xraph/warden/dsl"
)

// Source identifies where to read .warden files from. Exactly one of
// EmbedRoot, FS+Root, or Dir must be set.
type Source struct {
	// EmbedRoot uses the package-embedded FS at the given subpath
	// (e.g. embedSharedRoot or embedPlatformRoot). Most callers use
	// the convenience constructors SharedSource / PlatformSource.
	EmbedRoot string

	// FS + Root override the embedded FS with a caller-supplied one.
	// Useful for tests and custom deployments.
	FS   fs.FS
	Root string

	// Dir loads from the local filesystem. Takes precedence over the
	// embed FS when set. Used for the operator override path
	// (cfg.WardenDir).
	Dir string
}

// SharedSource returns the Source that points at the embedded files
// applied to every app.
func SharedSource() Source { return Source{EmbedRoot: embedSharedRoot} }

// PlatformSource returns the Source that points at the embedded files
// applied only to the platform app.
func PlatformSource() Source { return Source{EmbedRoot: embedPlatformRoot} }

// LoadOptions configures Load.
type LoadOptions struct {
	// AppID is substituted for ${APP_ID} placeholders in source files.
	// Required when sources contain `tenant ${APP_ID}` headers.
	AppID string

	// ExtraVars is merged into the variable map after AppID. Lets
	// callers add additional substitutions without losing AppID.
	ExtraVars map[string]string
}

// Load parses (and resolves) the .warden files from the given source,
// returning the merged DSL program. Diagnostics are returned as a
// non-nil error when any are present so callers can short-circuit on
// parse/resolve failures.
func Load(src Source, opts LoadOptions) (*dsl.Program, error) {
	loadOpts := []dsl.LoadOption{dsl.WithVariables(buildVars(opts))}

	prog, diags, err := loadFromSource(src, loadOpts)
	if err != nil {
		return nil, fmt.Errorf("wardenseed: load: %w", err)
	}
	if hasErrorDiagnostics(diags) {
		return nil, &dsl.DiagnosticError{Diags: diags}
	}
	if errs := dsl.Resolve(prog); hasErrorDiagnostics(errs) {
		return nil, &dsl.DiagnosticError{Diags: errs}
	}
	return prog, nil
}

func loadFromSource(src Source, opts []dsl.LoadOption) (*dsl.Program, []*dsl.Diagnostic, error) {
	switch {
	case src.Dir != "":
		if _, statErr := os.Stat(src.Dir); statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				return nil, nil, fmt.Errorf("dir %q does not exist", src.Dir)
			}
			return nil, nil, statErr
		}
		return dsl.LoadDir(src.Dir, opts...)

	case src.FS != nil:
		root := src.Root
		if root == "" {
			root = "."
		}
		return dsl.LoadFS(src.FS, root, opts...)

	case src.EmbedRoot != "":
		return dsl.LoadFS(FS, src.EmbedRoot, opts...)

	default:
		return nil, nil, errors.New("source has no Dir, FS, or EmbedRoot set")
	}
}

func buildVars(opts LoadOptions) dsl.Variables {
	vars := dsl.Variables{}
	if opts.AppID != "" {
		vars["APP_ID"] = opts.AppID
	}
	for k, v := range opts.ExtraVars {
		vars[k] = v
	}
	return vars
}

// hasErrorDiagnostics returns true if any diagnostic in the slice is
// non-warning severity. Today the parser/resolver don't tag severity, so
// we treat every diagnostic as an error — but the helper exists so we
// can soften that later without rewriting call sites.
func hasErrorDiagnostics(diags []*dsl.Diagnostic) bool {
	return len(diags) > 0
}

// FilesystemDir is a small convenience that resolves a relative dir
// against the current working directory. Used by extension.go to build
// the operator override Source from cfg.WardenDir.
func FilesystemDir(dir string) (Source, error) {
	if dir == "" {
		return Source{}, errors.New("wardenseed: empty dir")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Source{}, fmt.Errorf("wardenseed: resolve %q: %w", dir, err)
	}
	return Source{Dir: abs}, nil
}
