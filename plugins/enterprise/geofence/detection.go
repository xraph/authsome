package geofence

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xraph/authsome/internal/errs"
)

// DetectionProvider defines the interface for VPN/Proxy/Tor detection.
type DetectionProvider interface {
	Check(ctx context.Context, ip string) (*DetectionResult, error)
	Name() string
}

// DetectionResult represents VPN/proxy detection results.
type DetectionResult struct {
	IPAddress    string
	IsVPN        bool
	IsProxy      bool
	IsTor        bool
	IsDatacenter bool
	VPNProvider  string
	FraudScore   *float64
	Provider     string
}

// IPQSProvider implements DetectionProvider using IPQualityScore.
type IPQSProvider struct {
	apiKey     string
	strictness int     // 0-3
	minScore   float64 // 0-100
	client     *http.Client
}

// NewIPQSProvider creates a new IPQualityScore provider.
func NewIPQSProvider(apiKey string, strictness int, minScore float64) *IPQSProvider {
	return &IPQSProvider{
		apiKey:     apiKey,
		strictness: strictness,
		minScore:   minScore,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *IPQSProvider) Name() string {
	return "ipqs"
}

func (p *IPQSProvider) Check(ctx context.Context, ip string) (*DetectionResult, error) {
	url := fmt.Sprintf(
		"https://www.ipqualityscore.com/api/json/ip/%s/%s?strictness=%d",
		p.apiKey, ip, p.strictness,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query IPQS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("IPQS returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Success      bool    `json:"success"`
		FraudScore   float64 `json:"fraud_score"`
		Proxy        bool    `json:"proxy"`
		VPN          bool    `json:"vpn"`
		Tor          bool    `json:"tor"`
		ActiveVPN    bool    `json:"active_vpn"`
		ActiveTor    bool    `json:"active_tor"`
		RecentAbuse  bool    `json:"recent_abuse"`
		BotStatus    bool    `json:"bot_status"`
		ISP          string  `json:"ISP"`
		Organization string  `json:"organization"`
		ASN          int     `json:"ASN"`
		Host         string  `json:"host"`
		IsCrawler    bool    `json:"is_crawler"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if !result.Success {
		return nil, errs.InternalServerErrorWithMessage("IPQS query was not successful")
	}

	fraudScore := result.FraudScore

	return &DetectionResult{
		IPAddress:    ip,
		IsVPN:        result.VPN || result.ActiveVPN,
		IsProxy:      result.Proxy,
		IsTor:        result.Tor || result.ActiveTor,
		IsDatacenter: false, // IPQS doesn't explicitly provide this
		VPNProvider:  result.ISP,
		FraudScore:   &fraudScore,
		Provider:     "ipqs",
	}, nil
}

// ProxyCheckProvider implements DetectionProvider using proxycheck.io.
type ProxyCheckProvider struct {
	apiKey string
	client *http.Client
}

// NewProxyCheckProvider creates a new proxycheck.io provider.
func NewProxyCheckProvider(apiKey string) *ProxyCheckProvider {
	return &ProxyCheckProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *ProxyCheckProvider) Name() string {
	return "proxycheck"
}

func (p *ProxyCheckProvider) Check(ctx context.Context, ip string) (*DetectionResult, error) {
	url := fmt.Sprintf("https://proxycheck.io/v2/%s?key=%s&vpn=1&asn=1", ip, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query proxycheck: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("proxycheck returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// proxycheck returns a nested structure with the IP as key
	ipData, ok := result[ip].(map[string]any)
	if !ok {
		return nil, errs.InternalServerErrorWithMessage("unexpected response format from proxycheck")
	}

	proxy, _ := ipData["proxy"].(string)
	proxyType, _ := ipData["type"].(string)
	provider, _ := ipData["provider"].(string)

	isProxy := proxy == "yes"
	isVPN := proxyType == "VPN"
	isTor := proxyType == "TOR"

	return &DetectionResult{
		IPAddress:    ip,
		IsVPN:        isVPN,
		IsProxy:      isProxy,
		IsTor:        isTor,
		IsDatacenter: false,
		VPNProvider:  provider,
		Provider:     "proxycheck",
	}, nil
}

// VPNAPIProvider implements DetectionProvider using vpnapi.io.
type VPNAPIProvider struct {
	apiKey string
	client *http.Client
}

// NewVPNAPIProvider creates a new vpnapi.io provider.
func NewVPNAPIProvider(apiKey string) *VPNAPIProvider {
	return &VPNAPIProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (p *VPNAPIProvider) Name() string {
	return "vpnapi"
}

func (p *VPNAPIProvider) Check(ctx context.Context, ip string) (*DetectionResult, error) {
	url := fmt.Sprintf("https://vpnapi.io/api/%s?key=%s", ip, p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query vpnapi: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("vpnapi returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Security struct {
			VPN   bool `json:"vpn"`
			Proxy bool `json:"proxy"`
			Tor   bool `json:"tor"`
			Relay bool `json:"relay"`
		} `json:"security"`
		Network struct {
			Network                      string `json:"network"`
			AutonomousSystemNumber       string `json:"autonomous_system_number"`
			AutonomousSystemOrganization string `json:"autonomous_system_organization"`
		} `json:"network"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &DetectionResult{
		IPAddress:    ip,
		IsVPN:        result.Security.VPN,
		IsProxy:      result.Security.Proxy,
		IsTor:        result.Security.Tor,
		IsDatacenter: false,
		VPNProvider:  result.Network.AutonomousSystemOrganization,
		Provider:     "vpnapi",
	}, nil
}

// StaticDetectionProvider implements a simple rule-based detection
// Useful for testing or when external APIs are not available.
type StaticDetectionProvider struct {
	vpnIPs        map[string]bool
	proxyIPs      map[string]bool
	torIPs        map[string]bool
	datacenterIPs map[string]bool
}

// NewStaticDetectionProvider creates a new static detection provider.
func NewStaticDetectionProvider() *StaticDetectionProvider {
	return &StaticDetectionProvider{
		vpnIPs:        make(map[string]bool),
		proxyIPs:      make(map[string]bool),
		torIPs:        make(map[string]bool),
		datacenterIPs: make(map[string]bool),
	}
}

func (p *StaticDetectionProvider) Name() string {
	return "static"
}

func (p *StaticDetectionProvider) Check(ctx context.Context, ip string) (*DetectionResult, error) {
	return &DetectionResult{
		IPAddress:    ip,
		IsVPN:        p.vpnIPs[ip],
		IsProxy:      p.proxyIPs[ip],
		IsTor:        p.torIPs[ip],
		IsDatacenter: p.datacenterIPs[ip],
		Provider:     "static",
	}, nil
}

// AddVPN marks an IP as a VPN.
func (p *StaticDetectionProvider) AddVPN(ip string) {
	p.vpnIPs[ip] = true
}

// AddProxy marks an IP as a proxy.
func (p *StaticDetectionProvider) AddProxy(ip string) {
	p.proxyIPs[ip] = true
}

// AddTor marks an IP as Tor.
func (p *StaticDetectionProvider) AddTor(ip string) {
	p.torIPs[ip] = true
}

// AddDatacenter marks an IP as datacenter.
func (p *StaticDetectionProvider) AddDatacenter(ip string) {
	p.datacenterIPs[ip] = true
}
