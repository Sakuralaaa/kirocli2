package Utils

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"kilocli2api/Models"
)

const csvEnabledValue = "True"

type AdminAccountSnapshot struct {
	ID                  int    `json:"id"`
	ClientID            string `json:"client_id"`
	ClientSecretPreview string `json:"client_secret_preview"`
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
			ClientSecretPreview: maskToken(item.ClientSecret),
			RefreshTokenPreview: maskToken(item.Token),
			Active:              isActive,
			Disabled:            item.Disabled,
			ExpiresAt:           item.AccessToken.ExpiresAt,
		})
	}

	accountSource := os.Getenv("ACCOUNT_SOURCE")
	if accountSource == "" {
		accountSource = "manual"
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

func persistManualAccount(rt Models.RefreshToken) error {
	if csvPath != "" {
		csvMutex.Lock()
		defer csvMutex.Unlock()

		file, err := os.OpenFile(csvPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		w := csv.NewWriter(file)
		if err := w.Write([]string{csvEnabledValue, rt.Token, rt.ClientId, rt.ClientSecret}); err != nil {
			return err
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return err
		}
		return nil
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
	return saveAPIAccountsToJSON(accounts)
}

func nextAccountIDLocked() int {
	maxID := 0
	for _, item := range RefreshTokens {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	return maxID + 1
}

func ensureAccountIDsLocked() bool {
	changed := false
	used := map[int]bool{}
	nextID := 1
	for _, item := range RefreshTokens {
		if item.ID > 0 {
			used[item.ID] = true
			if item.ID >= nextID {
				nextID = item.ID + 1
			}
		}
	}
	seen := map[int]bool{}
	for idx, item := range RefreshTokens {
		if item.ID <= 0 || seen[item.ID] {
			for used[nextID] {
				nextID++
			}
			RefreshTokens[idx].ID = nextID
			used[nextID] = true
			nextID++
			changed = true
			continue
		}
		seen[item.ID] = true
		used[item.ID] = true
	}
	return changed
}

func persistAllAccountsToCSVLocked() error {
	header := []string{"enabled", "refresh_token", "client_id", "client_secret"}
	if data, err := os.ReadFile(csvPath); err == nil {
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
			parts := strings.Split(lines[0], ",")
			if len(parts) >= 4 {
				header = parts
			}
		}
	}

	file, err := os.OpenFile(csvPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err := w.Write(header); err != nil {
		return err
	}
	for _, item := range RefreshTokens {
		enabled := "True"
		if item.Disabled {
			enabled = "False"
		}
		if err := w.Write([]string{enabled, item.Token, item.ClientId, item.ClientSecret}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func persistAllAccountsToJSONLocked() error {
	accounts := make([]APIAccount, 0, len(RefreshTokens))
	for _, item := range RefreshTokens {
		accounts = append(accounts, APIAccount{
			ID:           item.ID,
			RefreshToken: item.Token,
			ClientID:     item.ClientId,
			ClientSecret: item.ClientSecret,
		})
	}
	return saveAPIAccountsToJSON(accounts)
}

func persistAllAccountsLocked() error {
	if csvPath != "" {
		return persistAllAccountsToCSVLocked()
	}
	return persistAllAccountsToJSONLocked()
}

func removeActiveTokenIndexLocked(target int) {
	next := make([]int, 0, len(ActiveTokens))
	for _, idx := range ActiveTokens {
		if idx != target {
			next = append(next, idx)
		}
	}
	ActiveTokens = next
}

func hasActiveTokenIndexLocked(target int) bool {
	for _, idx := range ActiveTokens {
		if idx == target {
			return true
		}
	}
	return false
}

func normalizeActiveTokenIndicesLocked(removed int) {
	for i, idx := range ActiveTokens {
		if idx > removed {
			ActiveTokens[i] = idx - 1
		}
	}
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
	if ensureAccountIDsLocked() {
		_ = persistAllAccountsLocked()
	}

	for _, item := range RefreshTokens {
		if item.Token == refreshToken {
			return AdminAccountSnapshot{}, fmt.Errorf("refresh token already exists")
		}
	}
	newEntry.ID = nextAccountIDLocked()

	RefreshTokens = append(RefreshTokens, newEntry)
	idx := len(RefreshTokens) - 1
	if activate {
		ActiveTokens = append(ActiveTokens, idx)
	}
	if err := persistManualAccount(newEntry); err != nil {
		RefreshTokens = RefreshTokens[:idx]
		if activate && len(ActiveTokens) > 0 {
			ActiveTokens = ActiveTokens[:len(ActiveTokens)-1]
		}
		return AdminAccountSnapshot{}, fmt.Errorf("failed to persist account: %w", err)
	}

	return AdminAccountSnapshot{
		ID:                  newEntry.ID,
		ClientID:            newEntry.ClientId,
		ClientSecretPreview: maskToken(newEntry.ClientSecret),
		RefreshTokenPreview: maskToken(newEntry.Token),
		Active:              activate,
		Disabled:            false,
		ExpiresAt:           newEntry.AccessToken.ExpiresAt,
	}, nil
}

func ListAdminAccounts() []AdminAccountSnapshot {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	if ensureAccountIDsLocked() {
		_ = persistAllAccountsLocked()
	}

	activeSet := map[int]bool{}
	for _, idx := range ActiveTokens {
		activeSet[idx] = true
	}

	accounts := make([]AdminAccountSnapshot, 0, len(RefreshTokens))
	for idx, item := range RefreshTokens {
		accounts = append(accounts, AdminAccountSnapshot{
			ID:                  item.ID,
			ClientID:            item.ClientId,
			ClientSecretPreview: maskToken(item.ClientSecret),
			RefreshTokenPreview: maskToken(item.Token),
			Active:              activeSet[idx] && !item.Disabled,
			Disabled:            item.Disabled,
			ExpiresAt:           item.AccessToken.ExpiresAt,
		})
	}

	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	return accounts
}

func UpdateAccountByID(id int, enabled *bool, clientID, clientSecret, refreshToken *string) (AdminAccountSnapshot, error) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	if ensureAccountIDsLocked() {
		_ = persistAllAccountsLocked()
	}

	targetIdx := -1
	for idx, item := range RefreshTokens {
		if item.ID == id {
			targetIdx = idx
			break
		}
	}
	if targetIdx < 0 {
		return AdminAccountSnapshot{}, fmt.Errorf("account not found")
	}

	if clientID != nil {
		RefreshTokens[targetIdx].ClientId = strings.TrimSpace(*clientID)
	}
	if clientSecret != nil {
		RefreshTokens[targetIdx].ClientSecret = strings.TrimSpace(*clientSecret)
	}
	if refreshToken != nil {
		newToken := strings.TrimSpace(*refreshToken)
		if newToken == "" {
			return AdminAccountSnapshot{}, fmt.Errorf("refresh token cannot be empty")
		}
		for idx, item := range RefreshTokens {
			if idx != targetIdx && item.Token == newToken {
				return AdminAccountSnapshot{}, fmt.Errorf("refresh token already exists")
			}
		}
		RefreshTokens[targetIdx].Token = newToken
	}
	if enabled != nil {
		RefreshTokens[targetIdx].Disabled = !*enabled
		if *enabled {
			if !hasActiveTokenIndexLocked(targetIdx) {
				ActiveTokens = append(ActiveTokens, targetIdx)
			}
		} else {
			removeActiveTokenIndexLocked(targetIdx)
		}
	}

	if err := persistAllAccountsLocked(); err != nil {
		return AdminAccountSnapshot{}, fmt.Errorf("failed to persist account changes: %w", err)
	}

	isActive := hasActiveTokenIndexLocked(targetIdx) && !RefreshTokens[targetIdx].Disabled
	item := RefreshTokens[targetIdx]
	return AdminAccountSnapshot{
		ID:                  item.ID,
		ClientID:            item.ClientId,
		ClientSecretPreview: maskToken(item.ClientSecret),
		RefreshTokenPreview: maskToken(item.Token),
		Active:              isActive,
		Disabled:            item.Disabled,
		ExpiresAt:           item.AccessToken.ExpiresAt,
	}, nil
}

func DeleteAccountByID(id int) error {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	if ensureAccountIDsLocked() {
		_ = persistAllAccountsLocked()
	}

	targetIdx := -1
	for idx, item := range RefreshTokens {
		if item.ID == id {
			targetIdx = idx
			break
		}
	}
	if targetIdx < 0 {
		return fmt.Errorf("account not found")
	}

	removeActiveTokenIndexLocked(targetIdx)
	RefreshTokens = append(RefreshTokens[:targetIdx], RefreshTokens[targetIdx+1:]...)
	normalizeActiveTokenIndicesLocked(targetIdx)

	if err := persistAllAccountsLocked(); err != nil {
		return fmt.Errorf("failed to persist account deletion: %w", err)
	}

	return nil
}

func BatchAddManualAccounts(records []map[string]interface{}) (int, []string) {
	success := 0
	errs := make([]string, 0)
	for idx, rec := range records {
		rt, _ := rec["refresh_token"].(string)
		cid, _ := rec["client_id"].(string)
		sec, _ := rec["client_secret"].(string)
		activate := false
		if raw, ok := rec["activate"]; ok {
			switch v := raw.(type) {
			case bool:
				activate = v
			case string:
				activate = strings.EqualFold(strings.TrimSpace(v), "true")
			}
		}
		if _, err := AddManualAccount(rt, cid, sec, activate); err != nil {
			errs = append(errs, fmt.Sprintf("line %d: %v", idx+1, err))
			continue
		}
		success++
	}
	return success, errs
}

func ParseAccountID(raw string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid account id")
	}
	return id, nil
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
