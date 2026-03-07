package social

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// discordEndpoint is the OAuth2 endpoint for Discord.
var discordEndpoint = oauth2.Endpoint{
	AuthURL:  "https://discord.com/oauth2/authorize",
	TokenURL: "https://discord.com/api/oauth2/token",
}

// discordProvider implements Provider for Discord OAuth2.
type discordProvider struct {
	config *oauth2.Config
}

// NewDiscordProvider creates a new Discord OAuth2 provider.
func NewDiscordProvider(cfg ProviderConfig) Provider {
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identify", "email"}
	}
	return &discordProvider{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       scopes,
			Endpoint:     discordEndpoint,
		},
	}
}

func (p *discordProvider) Name() string                 { return "discord" }
func (p *discordProvider) OAuth2Config() *oauth2.Config { return p.config }

func (p *discordProvider) FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, fmt.Errorf("social: discord: fetch user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("social: discord: fetch user: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Email         string `json:"email"`
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("social: discord: decode user: %w", err)
	}

	name := info.GlobalName
	if name == "" {
		name = info.Username
	}

	var avatarURL string
	if info.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", info.ID, info.Avatar)
	}

	return &ProviderUser{
		ProviderUserID: info.ID,
		Email:          info.Email,
		FirstName:      name,
		AvatarURL:      avatarURL,
	}, nil
}
