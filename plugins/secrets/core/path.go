package core

import (
	"regexp"
	"sort"
	"strings"
)

// Path validation constants
const (
	// MinPathLength is the minimum length for a secret path
	MinPathLength = 1
	// MaxPathLength is the maximum length for a secret path
	MaxPathLength = 512
	// MaxPathSegments is the maximum number of path segments
	MaxPathSegments = 20
	// PathSeparator is the separator used in secret paths
	PathSeparator = "/"
)

// pathRegex validates path format: alphanumeric with underscores, hyphens, and forward slashes
// Must start and end with alphanumeric character
// Examples: "database/postgres/password", "api-keys/stripe", "config_v2/settings"
var pathRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_\-/]*[a-zA-Z0-9])?$`)

// segmentRegex validates individual path segments
var segmentRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9_\-]*[a-zA-Z0-9])?$`)

// ParsePath parses a secret path into segments and extracts the key (leaf node)
// Returns the parent segments, the key name, and any error
func ParsePath(path string) (segments []string, key string, err error) {
	// Check for double slashes before normalization
	if strings.Contains(path, "//") {
		return nil, "", ErrInvalidPath(path, "path cannot contain consecutive slashes")
	}

	// Normalize path
	path = NormalizePath(path)

	if path == "" {
		return nil, "", ErrInvalidPath(path, "path cannot be empty")
	}

	if len(path) > MaxPathLength {
		return nil, "", ErrInvalidPath(path, "path exceeds maximum length")
	}

	// Validate path format
	if !pathRegex.MatchString(path) {
		return nil, "", ErrInvalidPath(path, "path contains invalid characters or format")
	}

	// Split into segments
	parts := strings.Split(path, PathSeparator)

	if len(parts) == 0 {
		return nil, "", ErrInvalidPath(path, "path cannot be empty")
	}

	if len(parts) > MaxPathSegments {
		return nil, "", ErrInvalidPath(path, "path exceeds maximum depth")
	}

	// Validate each segment
	for _, segment := range parts {
		if segment == "" {
			return nil, "", ErrInvalidPath(path, "path contains empty segments")
		}
		if !segmentRegex.MatchString(segment) {
			return nil, "", ErrInvalidPath(path, "segment '"+segment+"' contains invalid characters")
		}
	}

	// Extract key (last segment) and parent segments
	key = parts[len(parts)-1]
	if len(parts) > 1 {
		segments = parts[:len(parts)-1]
	} else {
		segments = []string{}
	}

	return segments, key, nil
}

// NormalizePath normalizes a secret path by:
// - Trimming leading/trailing slashes and whitespace
// - Converting to lowercase
// - Removing consecutive slashes
func NormalizePath(path string) string {
	// Trim whitespace
	path = strings.TrimSpace(path)

	// Trim leading/trailing slashes
	path = strings.Trim(path, PathSeparator)

	// Convert to lowercase
	path = strings.ToLower(path)

	// Remove consecutive slashes by splitting and rejoining
	parts := strings.Split(path, PathSeparator)
	cleanParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	return strings.Join(cleanParts, PathSeparator)
}

// GetParentPath returns the parent path (everything except the last segment)
// Returns empty string if the path has no parent
func GetParentPath(path string) string {
	path = NormalizePath(path)
	if path == "" {
		return ""
	}

	idx := strings.LastIndex(path, PathSeparator)
	if idx == -1 {
		return "" // No parent, this is a root-level key
	}

	return path[:idx]
}

// GetKey returns the key (last segment) from a path
func GetKey(path string) string {
	path = NormalizePath(path)
	if path == "" {
		return ""
	}

	idx := strings.LastIndex(path, PathSeparator)
	if idx == -1 {
		return path // Entire path is the key
	}

	return path[idx+1:]
}

// JoinPath joins path segments into a single path
func JoinPath(segments ...string) string {
	nonEmpty := make([]string, 0, len(segments))
	for _, s := range segments {
		s = strings.Trim(s, PathSeparator)
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	return NormalizePath(strings.Join(nonEmpty, PathSeparator))
}

// MatchesPrefix checks if a path matches a given prefix
// Both paths are normalized before comparison
func MatchesPrefix(path, prefix string) bool {
	path = NormalizePath(path)
	prefix = NormalizePath(prefix)

	if prefix == "" {
		return true // Empty prefix matches everything
	}

	if path == prefix {
		return true
	}

	// Check if path starts with prefix followed by separator
	return strings.HasPrefix(path, prefix+PathSeparator)
}

// IsValidPath checks if a path is valid without returning detailed errors
func IsValidPath(path string) bool {
	_, _, err := ParsePath(path)
	return err == nil
}

// GetDepth returns the depth (number of segments) of a path
func GetDepth(path string) int {
	path = NormalizePath(path)
	if path == "" {
		return 0
	}
	return strings.Count(path, PathSeparator) + 1
}

// GetAncestorPaths returns all ancestor paths for a given path
// Example: "a/b/c/d" returns ["a", "a/b", "a/b/c"]
func GetAncestorPaths(path string) []string {
	path = NormalizePath(path)
	if path == "" {
		return nil
	}

	parts := strings.Split(path, PathSeparator)
	if len(parts) <= 1 {
		return nil
	}

	ancestors := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		ancestors = append(ancestors, strings.Join(parts[:i], PathSeparator))
	}

	return ancestors
}

// PathToConfigKey converts a secret path to a config key format
// Example: "database/postgres/password" -> "database.postgres.password"
func PathToConfigKey(path string) string {
	path = NormalizePath(path)
	return strings.ReplaceAll(path, PathSeparator, ".")
}

// ConfigKeyToPath converts a config key to a secret path format
// Example: "database.postgres.password" -> "database/postgres/password"
func ConfigKeyToPath(configKey string) string {
	return NormalizePath(strings.ReplaceAll(configKey, ".", PathSeparator))
}

// BuildTree builds a tree structure from a list of paths
// Returns a map where keys are folder paths and values are lists of secret paths
func BuildTree(paths []string) map[string][]string {
	tree := make(map[string][]string)

	for _, path := range paths {
		path = NormalizePath(path)
		parent := GetParentPath(path)
		if parent == "" {
			parent = "/" // Root level
		}
		tree[parent] = append(tree[parent], path)
	}

	// Sort each folder's contents
	for folder := range tree {
		sort.Strings(tree[folder])
	}

	return tree
}

// ExtractFolders extracts unique folder paths from a list of secret paths
func ExtractFolders(paths []string) []string {
	folderSet := make(map[string]struct{})

	for _, path := range paths {
		// Get all ancestor paths
		ancestors := GetAncestorPaths(path)
		for _, ancestor := range ancestors {
			folderSet[ancestor] = struct{}{}
		}
	}

	folders := make([]string, 0, len(folderSet))
	for folder := range folderSet {
		folders = append(folders, folder)
	}

	sort.Strings(folders)
	return folders
}

// SortByPath sorts a slice of paths in natural order (folders before files at each level)
func SortByPath(paths []string) {
	sort.Slice(paths, func(i, j int) bool {
		pi := NormalizePath(paths[i])
		pj := NormalizePath(paths[j])

		// Split into segments
		si := strings.Split(pi, PathSeparator)
		sj := strings.Split(pj, PathSeparator)

		// Compare segment by segment
		minLen := len(si)
		if len(sj) < minLen {
			minLen = len(sj)
		}

		for k := 0; k < minLen; k++ {
			if si[k] != sj[k] {
				return si[k] < sj[k]
			}
		}

		// Shorter paths come first (folders before contents)
		return len(si) < len(sj)
	})
}
