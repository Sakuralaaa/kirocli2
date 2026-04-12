package Utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"kilocli2api/Models"
)

const csvEnabledValue = "True"

type AdminAccountSnapshot struct {
	ID                  int    `json:"id"`
	ClientID            string `json:"client_id"`
	RefreshTokenPreview string `json:"refresh_token_preview"`
	Active              bool   `json:"active"`
	Disabled            bool   `json:"disabled"`
	ExpiresAt           int64  `json:"expires_at"`
}

type AdminSnapshot struct {
	AccountSource        string                 `json:"account_source"`
	TotalAccounts        int                    `json:"total_accounts"`
	ActiveAccountCount   int                    `json:"active_account_count"`
	DisabledAccountCount int                    `json:"disabled_account_count"`
	ValidTokenCount      int                    `json:"valid_token_count"`
	Accounts             []AdminAccountSnapshot `json:"accounts"`
}

func maskToken(raw string) string {
	if len(raw) <= 8 {
		return "****"
	}
	return raw[:4] + "..." + raw[len(raw)-4:]
}

func GetAdminSnapshot() AdminSnapshot {
	tokenMutex.RLock()
	defer tokenMutex.RUnlock()

	activeSet := map[int]bool{}
	for _, idx := range ActiveTokens {
		activeSet[idx] = true
	}

	now := time.Now().Unix()
	accounts := make([]AdminAccountSnapshot, 0, len(RefreshTokens))
	disabledCount := 0
	validTokenCount := 0
	activeCount := 0

	for idx, item := range RefreshTokens {
		isActive := activeSet[idx] && !item.Disabled
		if isActive {
			activeCount++
		}
		if item.Disabled {
			disabledCount++
		}
		if item.AccessToken.ExpiresAt > now && !item.Disabled {
			validTokenCount++
		}

		accounts = append(accounts, AdminAccountSnapshot{
			ID:                  item.ID,
			ClientID:            item.ClientId,
			RefreshTokenPreview: maskToken(item.Token),
			Active:              isActive,
			Disabled:            item.Disabled,
			ExpiresAt:           item.AccessToken.ExpiresAt,
		})
	}

	accountSource := os.Getenv("ACCOUNT_SOURCE")
	if accountSource == "" {
		accountSource = "csv"
	}

	return AdminSnapshot{
		AccountSource:        accountSource,
		TotalAccounts:        len(RefreshTokens),
		ActiveAccountCount:   activeCount,
		DisabledAccountCount: disabledCount,
		ValidTokenCount:      validTokenCount,
		Accounts:             accounts,
	}
}

func persistManualAccount(rt Models.RefreshToken) {
	if csvPath != "" {
		csvMutex.Lock()
		defer csvMutex.Unlock()

		file, err := os.OpenFile(csvPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			NormalLogger.Printf("Failed to open CSV for append: %v\n", err)
			return
		}
		defer file.Close()

		w := csv.NewWriter(file)
		if err := w.Write([]string{csvEnabledValue, rt.Token, rt.ClientId, rt.ClientSecret}); err != nil {
			NormalLogger.Printf("Failed to append account into CSV: %v\n", err)
		}
		w.Flush()
		return
	}

	csvMutex.Lock()
	defer csvMutex.Unlock()

	accounts, err := loadAPIAccountsFromJSON()
	if err != nil {
		accounts = []APIAccount{}
	}
	accounts = append(accounts, APIAccount{
		ID:           rt.ID,
		RefreshToken: rt.Token,
		ClientID:     rt.ClientId,
		ClientSecret: rt.ClientSecret,
	})
	saveAPIAccountsToJSON(accounts)
}

func AddManualAccount(refreshToken, clientID, clientSecret string, activate bool) (AdminAccountSnapshot, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)

	if refreshToken == "" || clientID == "" || clientSecret == "" {
		return AdminAccountSnapshot{}, fmt.Errorf("refresh_token, client_id, client_secret are required")
	}

	newEntry := Models.RefreshToken{
		Token:        refreshToken,
		ClientId:     clientID,
		ClientSecret: clientSecret,
	}

	if activate {
		accessToken, err := GetAccessTokenFromRefreshToken(newEntry)
		if err != nil {
			return AdminAccountSnapshot{}, err
		}
		newEntry.AccessToken = accessToken
	}

	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	for _, item := range RefreshTokens {
		if item.Token == refreshToken {
			return AdminAccountSnapshot{}, fmt.Errorf("refresh token already exists")
		}
	}

	RefreshTokens = append(RefreshTokens, newEntry)
	idx := len(RefreshTokens) - 1
	if activate {
		ActiveTokens = append(ActiveTokens, idx)
	}
	persistManualAccount(newEntry)

	return AdminAccountSnapshot{
		ID:                  newEntry.ID,
		ClientID:            newEntry.ClientId,
		RefreshTokenPreview: maskToken(newEntry.Token),
		Active:              activate,
		Disabled:            false,
		ExpiresAt:           newEntry.AccessToken.ExpiresAt,
	}, nil
}

func RefreshAllActiveTokensNow() (int, int) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	refreshed := 0
	failed := 0
	for _, idx := range ActiveTokens {
		if idx < 0 || idx >= len(RefreshTokens) || RefreshTokens[idx].Disabled {
			continue
		}
		newToken, err := GetAccessTokenFromRefreshToken(RefreshTokens[idx])
		if err != nil {
			failed++
			continue
		}
		RefreshTokens[idx].AccessToken = newToken
		refreshed++
	}

	return refreshed, failed
}
