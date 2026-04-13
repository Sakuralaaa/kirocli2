package KiroAuth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RefreshInput struct {
	RefreshToken string
	ClientID     string
	ClientSecret string
	AuthMethod   string
	Region       string
}

func RefreshToken(input RefreshInput) (string, string, int64, error) {
	if input.AuthMethod == "social" {
		accessToken, refreshToken, expiresAt, err := refreshSocialToken(input.RefreshToken)
		if err == nil {
			return accessToken, refreshToken, expiresAt, nil
		}
		if strings.TrimSpace(input.ClientID) != "" && strings.TrimSpace(input.ClientSecret) != "" {
			oidcAccess, oidcRefresh, oidcExpires, oidcErr := refreshOIDCToken(input.RefreshToken, input.ClientID, input.ClientSecret, input.Region)
			if oidcErr == nil {
				return oidcAccess, oidcRefresh, oidcExpires, nil
			}
			return "", "", 0, fmt.Errorf("social refresh failed: %v; oidc fallback failed: %v", err, oidcErr)
		}
		return "", "", 0, err
	}
	accessToken, refreshToken, expiresAt, err := refreshOIDCToken(input.RefreshToken, input.ClientID, input.ClientSecret, input.Region)
	if err == nil {
		return accessToken, refreshToken, expiresAt, nil
	}
	socialAccess, socialRefresh, socialExpires, socialErr := refreshSocialToken(input.RefreshToken)
	if socialErr == nil {
		return socialAccess, socialRefresh, socialExpires, nil
	}
	return "", "", 0, fmt.Errorf("oidc refresh failed: %v; social fallback failed: %v", err, socialErr)
}

func refreshOIDCToken(refreshToken, clientID, clientSecret, region string) (string, string, int64, error) {
	if clientID == "" || clientSecret == "" {
		return "", "", 0, fmt.Errorf("OIDC refresh requires clientId and clientSecret")
	}
	normalizedRegion, err := normalizeRegion(region)
	if err != nil {
		return "", "", 0, err
	}

	url := fmt.Sprintf("https://oidc.%s.amazonaws.com/token", normalizedRegion)
	payload := map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"refreshToken": refreshToken,
		"grantType":    "refresh_token",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", 0, fmt.Errorf("refresh failed: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int    `json:"expiresIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", 0, err
	}
	expiresAt := time.Now().Unix() + int64(result.ExpiresIn)
	return result.AccessToken, result.RefreshToken, expiresAt, nil
}

func refreshSocialToken(refreshToken string) (string, string, int64, error) {
	url := "https://prod.us-east-1.auth.desktop.kiro.dev/refreshToken"
	payload := map[string]string{"refreshToken": refreshToken}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", "", 0, fmt.Errorf("refresh failed: %d %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ExpiresIn    int    `json:"expiresIn"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", 0, err
	}
	expiresAt := time.Now().Unix() + int64(result.ExpiresIn)
	return result.AccessToken, result.RefreshToken, expiresAt, nil
}
