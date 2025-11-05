package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GeoProvider defines the interface for geolocation providers
type GeoProvider interface {
	Lookup(ctx context.Context, ip string) (*GeoData, error)
	Name() string
}

// GeoData represents geolocation information
type GeoData struct {
	IPAddress    string
	Country      string
	CountryCode  string // ISO 3166-1 alpha-2
	Region       string
	City         string
	Latitude     *float64
	Longitude    *float64
	AccuracyKm   *float64
	ASN          string
	ISP          string
	Organization string
	Provider     string
}

// MaxMindProvider implements GeoProvider using MaxMind GeoIP2
type MaxMindProvider struct {
	licenseKey   string
	databasePath string
	client       *http.Client
}

// NewMaxMindProvider creates a new MaxMind provider
func NewMaxMindProvider(licenseKey, databasePath string) *MaxMindProvider {
	return &MaxMindProvider{
		licenseKey:   licenseKey,
		databasePath: databasePath,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *MaxMindProvider) Name() string {
	return "maxmind"
}

func (p *MaxMindProvider) Lookup(ctx context.Context, ip string) (*GeoData, error) {
	// If database path is provided, use local database
	// Otherwise, use MaxMind web service API
	if p.databasePath != "" {
		return p.lookupLocal(ctx, ip)
	}
	return p.lookupAPI(ctx, ip)
}

func (p *MaxMindProvider) lookupAPI(ctx context.Context, ip string) (*GeoData, error) {
	url := fmt.Sprintf("https://geoip.maxmind.com/geoip/v2.1/city/%s", ip)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(p.licenseKey, "")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query MaxMind API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MaxMind API returned status %d", resp.StatusCode)
	}

	var result struct {
		City struct {
			Names map[string]string `json:"names"`
		} `json:"city"`
		Country struct {
			ISOCode string            `json:"iso_code"`
			Names   map[string]string `json:"names"`
		} `json:"country"`
		Location struct {
			Latitude       float64 `json:"latitude"`
			Longitude      float64 `json:"longitude"`
			AccuracyRadius int     `json:"accuracy_radius"`
		} `json:"location"`
		Subdivisions []struct {
			Names map[string]string `json:"names"`
		} `json:"subdivisions"`
		Traits struct {
			AutonomousSystemNumber       int    `json:"autonomous_system_number"`
			AutonomousSystemOrganization string `json:"autonomous_system_organization"`
			ISP                          string `json:"isp"`
			Organization                 string `json:"organization"`
		} `json:"traits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	lat := result.Location.Latitude
	lon := result.Location.Longitude
	accuracyKm := float64(result.Location.AccuracyRadius)

	region := ""
	if len(result.Subdivisions) > 0 {
		region = result.Subdivisions[0].Names["en"]
	}

	return &GeoData{
		IPAddress:    ip,
		Country:      result.Country.Names["en"],
		CountryCode:  result.Country.ISOCode,
		Region:       region,
		City:         result.City.Names["en"],
		Latitude:     &lat,
		Longitude:    &lon,
		AccuracyKm:   &accuracyKm,
		ASN:          fmt.Sprintf("AS%d", result.Traits.AutonomousSystemNumber),
		ISP:          result.Traits.ISP,
		Organization: result.Traits.Organization,
		Provider:     "maxmind",
	}, nil
}

func (p *MaxMindProvider) lookupLocal(ctx context.Context, ip string) (*GeoData, error) {
	// TODO: Implement local MaxMind database lookup using mmdb reader
	// For now, fallback to API
	return p.lookupAPI(ctx, ip)
}

// IPAPIProvider implements GeoProvider using ipapi.com
type IPAPIProvider struct {
	apiKey string
	client *http.Client
}

// NewIPAPIProvider creates a new ipapi.com provider
func NewIPAPIProvider(apiKey string) *IPAPIProvider {
	return &IPAPIProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *IPAPIProvider) Name() string {
	return "ipapi"
}

func (p *IPAPIProvider) Lookup(ctx context.Context, ip string) (*GeoData, error) {
	url := fmt.Sprintf("https://api.ipapi.com/%s?access_key=%s", ip, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query ipapi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ipapi returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Country     string  `json:"country_name"`
		CountryCode string  `json:"country_code"`
		Region      string  `json:"region_name"`
		City        string  `json:"city"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Connection  struct {
			ASN int    `json:"asn"`
			ISP string `json:"isp"`
		} `json:"connection"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	lat := result.Latitude
	lon := result.Longitude

	return &GeoData{
		IPAddress:    ip,
		Country:      result.Country,
		CountryCode:  result.CountryCode,
		Region:       result.Region,
		City:         result.City,
		Latitude:     &lat,
		Longitude:    &lon,
		ASN:          fmt.Sprintf("AS%d", result.Connection.ASN),
		ISP:          result.Connection.ISP,
		Organization: result.Connection.ISP,
		Provider:     "ipapi",
	}, nil
}

// IPInfoProvider implements GeoProvider using ipinfo.io
type IPInfoProvider struct {
	token  string
	client *http.Client
}

// NewIPInfoProvider creates a new ipinfo.io provider
func NewIPInfoProvider(token string) *IPInfoProvider {
	return &IPInfoProvider{
		token: token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *IPInfoProvider) Name() string {
	return "ipinfo"
}

func (p *IPInfoProvider) Lookup(ctx context.Context, ip string) (*GeoData, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s?token=%s", ip, p.token)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query ipinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ipinfo returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Country string `json:"country"`
		Region  string `json:"region"`
		City    string `json:"city"`
		Loc     string `json:"loc"` // "lat,lon"
		Org     string `json:"org"` // "AS12345 ISP Name"
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse location
	var lat, lon float64
	fmt.Sscanf(result.Loc, "%f,%f", &lat, &lon)

	// Parse ASN and org
	var asn string
	var org string
	fmt.Sscanf(result.Org, "%s %s", &asn, &org)

	return &GeoData{
		IPAddress:    ip,
		Country:      "", // ipinfo doesn't provide full country name in basic response
		CountryCode:  result.Country,
		Region:       result.Region,
		City:         result.City,
		Latitude:     &lat,
		Longitude:    &lon,
		ASN:          asn,
		Organization: org,
		ISP:          org,
		Provider:     "ipinfo",
	}, nil
}

// IPGeolocationProvider implements GeoProvider using ipgeolocation.io
type IPGeolocationProvider struct {
	apiKey string
	client *http.Client
}

// NewIPGeolocationProvider creates a new ipgeolocation.io provider
func NewIPGeolocationProvider(apiKey string) *IPGeolocationProvider {
	return &IPGeolocationProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *IPGeolocationProvider) Name() string {
	return "ipgeolocation"
}

func (p *IPGeolocationProvider) Lookup(ctx context.Context, ip string) (*GeoData, error) {
	url := fmt.Sprintf("https://api.ipgeolocation.io/ipgeo?apiKey=%s&ip=%s", p.apiKey, ip)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query ipgeolocation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ipgeolocation returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		CountryName  string `json:"country_name"`
		CountryCode  string `json:"country_code2"`
		State        string `json:"state_prov"`
		City         string `json:"city"`
		Latitude     string `json:"latitude"`
		Longitude    string `json:"longitude"`
		ISP          string `json:"isp"`
		Organization string `json:"organization"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var lat, lon float64
	fmt.Sscanf(result.Latitude, "%f", &lat)
	fmt.Sscanf(result.Longitude, "%f", &lon)

	return &GeoData{
		IPAddress:    ip,
		Country:      result.CountryName,
		CountryCode:  result.CountryCode,
		Region:       result.State,
		City:         result.City,
		Latitude:     &lat,
		Longitude:    &lon,
		ISP:          result.ISP,
		Organization: result.Organization,
		Provider:     "ipgeolocation",
	}, nil
}
