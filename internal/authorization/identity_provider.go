package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"donetick.com/core/config"
	"golang.org/x/oauth2"
)

type IdentityProviderUserInfo struct {
	Identifier  string
	DisplayName string
	Email       string
}

type IdentityProvider struct {
	config    *config.OAuth2Config
	isEnabled bool
}

func NewIdentityProvider(cfg *config.Config) *IdentityProvider {
	if cfg.OAuth2Config.ClientID == "" || cfg.OAuth2Config.ClientSecret == "" {
		return &IdentityProvider{isEnabled: false}
	}
	return &IdentityProvider{config: &cfg.OAuth2Config, isEnabled: true}
}

func (i *IdentityProvider) ExchangeToken(ctx context.Context, code string) (string, error) {
	if !i.isEnabled {
		return "", errors.New("identity provider is not enabled")
	}

	conf := &oauth2.Config{
		ClientID:     i.config.ClientID,
		ClientSecret: i.config.ClientSecret,
		RedirectURL:  i.config.RedirectURL,
		Scopes:       i.config.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  i.config.AuthURL,
			TokenURL: i.config.TokenURL,
		},
	}
	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return "", err
	}

	accessToken, ok := token.AccessToken, token.Valid()
	if !ok {
		return "", errors.New("access token not found")
	}

	return accessToken, nil
}

func (i *IdentityProvider) GetUserInfo(ctx context.Context, accessToken string) (*IdentityProviderUserInfo, error) {
	req, err := http.NewRequest("GET", i.config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var claims map[string]any
	err = json.Unmarshal(body, &claims)
	if err != nil {
		return nil, errors.New("failed to unmarshal claims")
	}
	userInfo := IdentityProviderUserInfo{}
	if val, ok := claims["sub"]; ok {
		userInfo.Identifier = val.(string)
	}
	if val, ok := claims["name"]; ok {
		userInfo.DisplayName = val.(string)
	}
	if val, ok := claims["email"]; ok {
		userInfo.Email = val.(string)
	}
	return &userInfo, nil
}
