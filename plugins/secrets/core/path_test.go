package core

import (
	"reflect"
	"testing"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantSegments []string
		wantKey      string
		wantErr      bool
	}{
		{
			name:         "simple path",
			path:         "database/password",
			wantSegments: []string{"database"},
			wantKey:      "password",
			wantErr:      false,
		},
		{
			name:         "deep path",
			path:         "database/postgres/credentials/password",
			wantSegments: []string{"database", "postgres", "credentials"},
			wantKey:      "password",
			wantErr:      false,
		},
		{
			name:         "single key",
			path:         "password",
			wantSegments: []string{},
			wantKey:      "password",
			wantErr:      false,
		},
		{
			name:         "with leading slash",
			path:         "/database/password",
			wantSegments: []string{"database"},
			wantKey:      "password",
			wantErr:      false,
		},
		{
			name:         "with trailing slash",
			path:         "database/password/",
			wantSegments: []string{"database"},
			wantKey:      "password",
			wantErr:      false,
		},
		{
			name:         "with underscores",
			path:         "api_keys/stripe_key",
			wantSegments: []string{"api_keys"},
			wantKey:      "stripe_key",
			wantErr:      false,
		},
		{
			name:         "with hyphens",
			path:         "api-keys/stripe-key",
			wantSegments: []string{"api-keys"},
			wantKey:      "stripe-key",
			wantErr:      false,
		},
		{
			name:         "uppercase normalized",
			path:         "DATABASE/PASSWORD",
			wantSegments: []string{"database"},
			wantKey:      "password",
			wantErr:      false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "only slash",
			path:    "/",
			wantErr: true,
		},
		{
			name:    "invalid characters",
			path:    "data@base/pass!word",
			wantErr: true,
		},
		{
			name:    "double slash",
			path:    "database//password",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, key, err := ParsePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePath() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(segments, tt.wantSegments) {
					t.Errorf("ParsePath() segments = %v, want %v", segments, tt.wantSegments)
				}

				if key != tt.wantKey {
					t.Errorf("ParsePath() key = %v, want %v", key, tt.wantKey)
				}
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple", "database/password", "database/password"},
		{"leading slash", "/database/password", "database/password"},
		{"trailing slash", "database/password/", "database/password"},
		{"both slashes", "/database/password/", "database/password"},
		{"uppercase", "DATABASE/PASSWORD", "database/password"},
		{"whitespace", "  database/password  ", "database/password"},
		{"double slash", "database//password", "database/password"},
		{"mixed", "  /DATABASE//PASSWORD/  ", "database/password"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizePath(tt.path); got != tt.want {
				t.Errorf("NormalizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"deep path", "a/b/c/d", "a/b/c"},
		{"two levels", "a/b", "a"},
		{"single key", "a", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetParentPath(tt.path); got != tt.want {
				t.Errorf("GetParentPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetKey(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"deep path", "a/b/c/d", "d"},
		{"two levels", "a/b", "b"},
		{"single key", "a", "a"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetKey(tt.path); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		segments []string
		want     string
	}{
		{"multiple segments", []string{"a", "b", "c"}, "a/b/c"},
		{"with slashes", []string{"/a/", "/b/", "/c/"}, "a/b/c"},
		{"with empty", []string{"a", "", "c"}, "a/c"},
		{"single", []string{"a"}, "a"},
		{"empty", []string{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JoinPath(tt.segments...); got != tt.want {
				t.Errorf("JoinPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		prefix string
		want   bool
	}{
		{"exact match", "a/b/c", "a/b/c", true},
		{"prefix match", "a/b/c/d", "a/b", true},
		{"no match", "a/b/c", "x/y", false},
		{"empty prefix", "a/b/c", "", true},
		{"partial segment", "a/bc", "a/b", false},
		{"same level", "a/b", "a/c", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchesPrefix(tt.path, tt.prefix); got != tt.want {
				t.Errorf("MatchesPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"valid simple", "database/password", true},
		{"valid deep", "a/b/c/d/e", true},
		{"valid single", "password", true},
		{"valid underscore", "api_key", true},
		{"valid hyphen", "api-key", true},
		{"invalid empty", "", false},
		{"invalid chars", "data@base", false},
		{"invalid space", "data base", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPath(tt.path); got != tt.want {
				t.Errorf("IsValidPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDepth(t *testing.T) {
	tests := []struct {
		name string
		path string
		want int
	}{
		{"single", "a", 1},
		{"two levels", "a/b", 2},
		{"deep", "a/b/c/d/e", 5},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDepth(tt.path); got != tt.want {
				t.Errorf("GetDepth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAncestorPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{"deep", "a/b/c/d", []string{"a", "a/b", "a/b/c"}},
		{"two levels", "a/b", []string{"a"}},
		{"single", "a", nil},
		{"empty", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetAncestorPaths(tt.path)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAncestorPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPathToConfigKey(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple", "database/password", "database.password"},
		{"deep", "a/b/c/d", "a.b.c.d"},
		{"single", "key", "key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathToConfigKey(tt.path); got != tt.want {
				t.Errorf("PathToConfigKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigKeyToPath(t *testing.T) {
	tests := []struct {
		name      string
		configKey string
		want      string
	}{
		{"simple", "database.password", "database/password"},
		{"deep", "a.b.c.d", "a/b/c/d"},
		{"single", "key", "key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConfigKeyToPath(tt.configKey); got != tt.want {
				t.Errorf("ConfigKeyToPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractFolders(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  []string
	}{
		{
			name:  "nested paths",
			paths: []string{"a/b/c", "a/b/d", "x/y"},
			want:  []string{"a", "a/b", "x"},
		},
		{
			name:  "single level",
			paths: []string{"a", "b"},
			want:  []string{},
		},
		{
			name:  "deep",
			paths: []string{"a/b/c/d/e"},
			want:  []string{"a", "a/b", "a/b/c", "a/b/c/d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFolders(tt.paths)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractFolders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkParsePath(b *testing.B) {
	path := "database/postgres/credentials/password"
	for b.Loop() {
		ParsePath(path)
	}
}

func BenchmarkNormalizePath(b *testing.B) {
	path := "  /DATABASE//PASSWORD/  "
	for b.Loop() {
		NormalizePath(path)
	}
}
