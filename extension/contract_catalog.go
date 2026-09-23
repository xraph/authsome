package extension

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	contract "github.com/xraph/forge/extensions/dashboard/contract"
	"github.com/xraph/forge/extensions/dashboard/contract/remote"
)

func fetchContractCatalog(ctx context.Context, baseURL, apiKey string) ([]*contract.ContractManifest, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+remote.DefaultManifestPath, nil)
	if err != nil {
		return nil, fmt.Errorf("build manifest request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+apiKey)
	}
	client := &http.Client{Timeout: remote.DefaultTimeout}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest catalog: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("manifest endpoint returned HTTP %d", response.StatusCode)
	}
	const maxCatalogBytes = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxCatalogBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read manifest catalog: %w", err)
	}
	if len(body) > maxCatalogBytes {
		return nil, fmt.Errorf("manifest catalog exceeds %d bytes", maxCatalogBytes)
	}
	var catalog struct {
		Manifests []*contract.ContractManifest `json:"manifests"`
	}
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("decode manifest catalog: %w", err)
	}
	if catalog.Manifests == nil {
		var manifest contract.ContractManifest
		if err := json.Unmarshal(body, &manifest); err != nil {
			return nil, fmt.Errorf("decode manifest: %w", err)
		}
		catalog.Manifests = []*contract.ContractManifest{&manifest}
	}
	if len(catalog.Manifests) == 0 {
		return nil, fmt.Errorf("manifest catalog is empty")
	}
	seen := make(map[string]bool, len(catalog.Manifests))
	for _, manifest := range catalog.Manifests {
		if manifest == nil || manifest.Contributor.Name == "" {
			return nil, fmt.Errorf("manifest is missing contributor.name")
		}
		if seen[manifest.Contributor.Name] {
			return nil, fmt.Errorf("duplicate contributor %q", manifest.Contributor.Name)
		}
		seen[manifest.Contributor.Name] = true
	}
	if !seen["auth"] {
		return nil, fmt.Errorf("manifest catalog is missing auth")
	}
	return catalog.Manifests, nil
}
