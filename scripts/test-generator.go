// +build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	
	"github.com/xraph/authsome/internal/clients/generator"
)

func main() {
	// Get workspace root
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	
	// Navigate to authsome root if needed
	root := wd
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			log.Fatal("Could not find authsome root")
		}
		root = parent
	}
	
	manifestDir := filepath.Join(root, "internal/clients/manifest/data")
	outputDir := filepath.Join(root, "clients")
	moduleName := "github.com/xraph/authsome/clients/go"
	
	fmt.Println("🔧 Generating Go client SDK...")
	fmt.Printf("  Manifest dir: %s\n", manifestDir)
	fmt.Printf("  Output dir: %s\n", outputDir)
	fmt.Printf("  Module name: %s\n", moduleName)
	
	gen, err := generator.NewGenerator(manifestDir, outputDir, moduleName)
	if err != nil {
		log.Fatalf("Failed to create generator: %v", err)
	}
	
	if err := gen.Generate(generator.LanguageGo, nil); err != nil {
		log.Fatalf("Failed to generate client: %v", err)
	}
	
	fmt.Println("✅ Go client generated successfully!")
}

