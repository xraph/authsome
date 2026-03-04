// Command authsome-sdkgen generates typed client SDKs from AuthSome API metadata.
//
// Usage:
//
//	go run ./sdkgen/cmd generate --lang=typescript --out=./sdk
//	go run ./sdkgen/cmd generate --lang=go --out=./sdk
//	go run ./sdkgen/cmd generate --lang=all --out=./sdk
//
// Flags:
//
//	--lang              Target language: "typescript", "go", or "all" (default: "all")
//	--out               Output directory (default: "./sdk")
//	--plugins           Comma-separated list of enabled plugins (default: "")
//	--title             API title for the spec (default: "AuthSome API")
//	--version           API version (default: "0.5.0")
//	--server            Server URL (default: "")
//	--output-mode       Output mode: "standalone" or "embedded" (default: "standalone")
//	--method-overrides  Comma-separated operationID=name overrides (e.g., "refreshSession=refresh,deleteAccount=deleteMe")
//	--from-spec         Path to a pre-generated OpenAPI spec JSON file (skip internal generation)
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/sdkgen/golang"
	"github.com/xraph/authsome/sdkgen/openapi"
	"github.com/xraph/authsome/sdkgen/typescript"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "generate" {
		fmt.Fprintln(os.Stderr, "Usage: authsome-sdkgen generate [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Generates typed client SDKs from AuthSome API metadata.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		fmt.Fprintln(os.Stderr, "  --lang              Target language: typescript, go, or all (default: all)")
		fmt.Fprintln(os.Stderr, "  --out               Output directory (default: ./sdk)")
		fmt.Fprintln(os.Stderr, "  --plugins           Comma-separated enabled plugins (default: \"\")")
		fmt.Fprintln(os.Stderr, "  --title             API title (default: AuthSome API)")
		fmt.Fprintln(os.Stderr, "  --version           API version (default: 0.5.0)")
		fmt.Fprintln(os.Stderr, "  --server            Server URL (optional)")
		fmt.Fprintln(os.Stderr, "  --output-mode       Output mode: standalone or embedded (default: standalone)")
		fmt.Fprintln(os.Stderr, "  --method-overrides  operationID=name overrides (e.g., refreshSession=refresh)")
		fmt.Fprintln(os.Stderr, "  --from-spec         Path to pre-generated OpenAPI spec JSON (skip internal generation)")
		os.Exit(1)
	}

	// Parse flags after the "generate" subcommand
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	lang := fs.String("lang", "all", "Target language: typescript, go, or all")
	outDir := fs.String("out", "./sdk", "Output directory")
	plugins := fs.String("plugins", "", "Comma-separated list of enabled plugins")
	title := fs.String("title", "AuthSome API", "API title")
	version := fs.String("version", "0.5.0", "API version")
	server := fs.String("server", "", "Server URL")
	outputMode := fs.String("output-mode", "standalone", "Output mode: standalone or embedded")
	methodOverrides := fs.String("method-overrides", "", "Comma-separated operationID=name overrides")
	fromSpec := fs.String("from-spec", "", "Path to pre-generated OpenAPI spec JSON (skip internal generation)")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Parse enabled plugins
	var enabledPlugins []string
	if *plugins != "" {
		enabledPlugins = strings.Split(*plugins, ",")
		for i, p := range enabledPlugins {
			enabledPlugins[i] = strings.TrimSpace(p)
		}
	}

	// Parse method overrides
	overrides := make(map[string]string)
	if *methodOverrides != "" {
		for _, pair := range strings.Split(*methodOverrides, ",") {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				overrides[parts[0]] = parts[1]
			}
		}
	}

	// Load or generate OpenAPI spec.
	var spec *openapi.Spec
	if *fromSpec != "" {
		fmt.Printf("Loading OpenAPI spec from %s...\n", *fromSpec)
		var err error
		spec, err = openapi.LoadSpecFromFile(*fromSpec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading spec: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Generating OpenAPI spec (hardcoded)...")
		specGen := openapi.NewGenerator(openapi.GeneratorConfig{
			Title:          *title,
			Version:        *version,
			ServerURL:      *server,
			EnabledPlugins: enabledPlugins,
		})
		spec = specGen.Generate()
	}

	// Generate SDKs
	switch strings.ToLower(*lang) {
	case "typescript", "ts":
		if err := generateTypeScript(spec, *outDir, *outputMode, overrides); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating TypeScript SDK: %v\n", err)
			os.Exit(1)
		}
	case "go", "golang":
		if err := generateGo(spec, *outDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating Go SDK: %v\n", err)
			os.Exit(1)
		}
	case "all":
		if err := generateTypeScript(spec, filepath.Join(*outDir, "typescript"), *outputMode, overrides); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating TypeScript SDK: %v\n", err)
			os.Exit(1)
		}
		if err := generateGo(spec, filepath.Join(*outDir, "go")); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating Go SDK: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown language: %s (supported: typescript, go, all)\n", *lang)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func generateTypeScript(spec *openapi.Spec, outDir, outputMode string, overrides map[string]string) error {
	fmt.Printf("Generating TypeScript SDK -> %s (mode: %s)\n", outDir, outputMode)

	gen := typescript.NewGenerator(typescript.GeneratorConfig{
		OutputMode:      outputMode,
		MethodOverrides: overrides,
	})
	files, err := gen.Generate(spec)
	if err != nil {
		return err
	}

	return writeFiles(outDir, files)
}

func generateGo(spec *openapi.Spec, outDir string) error {
	fmt.Printf("Generating Go SDK -> %s\n", outDir)

	gen := golang.NewGenerator(golang.GeneratorConfig{})
	files, err := gen.Generate(spec)
	if err != nil {
		return err
	}

	return writeGoFiles(outDir, files)
}

func writeFiles(outDir string, files []typescript.GeneratedFile) error {
	for _, f := range files {
		path := filepath.Join(outDir, f.Path)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
		if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Printf("  wrote %s\n", f.Path)
	}
	return nil
}

func writeGoFiles(outDir string, files []golang.GeneratedFile) error {
	for _, f := range files {
		path := filepath.Join(outDir, f.Path)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
		if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		fmt.Printf("  wrote %s\n", f.Path)
	}
	return nil
}
