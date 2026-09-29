package mask

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every log field that names an address or a phone number goes through a
// mask. This walks the module so a new site cannot slip in unmasked.
func TestLogFieldsAreMasked(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	field := regexp.MustCompile(`log\.String\("(email|phone|identifier|to|recipient)",\s*([^)]*\))`)
	var offenders []string
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == "sdk" || name == "docs" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range field.FindAllStringSubmatch(string(src), -1) {
			value := m[2]
			if strings.Contains(value, "mask.") || strings.Contains(value, "Location") {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			offenders = append(offenders, rel+": "+m[0])
		}
		return nil
	})
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	if len(offenders) > 0 {
		t.Errorf("log fields carrying an unmasked identifier:\n  %s", strings.Join(offenders, "\n  "))
	}
}
