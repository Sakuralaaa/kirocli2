package Utils

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"kilocli2api/Models"
)

var initDoOnce = &sync.Once{}

var RefreshTokens []Models.RefreshToken
var ActiveTokens []int
var tokenMutex sync.RWMutex
var csvMutex sync.Mutex
var tokenIndex int
var csvPath string
var apiAccountsPath = "data/api_accounts.json"
var activeTokenCount int
var nextRefreshTokenIndex int
var maxRefreshAttempt int

func getProxyTransport() *http.Transport {
	return GetProxyTransport()
}

func loadAccountsFromCSV(csvPath string) error {
	if _, err := os.Stat(csvPath); err != nil {
		return fmt.Errorf("csv file unavailable: %w", err)
	}
	file, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV file: %w", err)
	}

	for i, record := range records {
		if i == 0 || len(record) < 4 {
			continue
		}
		if strings.TrimSpace(record[0]) == "True" {
			RefreshTokens = append(RefreshTokens, Models.RefreshToken{
				Token:        strings.TrimSpace(record[1]),
				ClientId:     strings.TrimSpace(record[2]),
				ClientSecret: strings.TrimSpace(record[3]),
			})
		}
	}

	if len(RefreshTokens) == 0 {
		return fmt.Errorf("no enabled accounts found in csv")
	}
	return nil
}

type APIAccount struct {
	ID           int    `json:"id"`
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type APIAccountResponse struct {
	ID   int    `json:"id"`
	Data string `json:"data"`
}

type APIAccountData struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func saveAPIAccountsToJSON(accounts []APIAccount) error {
	if accounts == nil {
		accounts = []APIAccount{}
	}
	if err := os.MkdirAll(filepath.Dir(apiAccountsPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(apiAccountsPath, data, 0o600); err != nil {
		return err
	}
	return nil
}

func loadAPIAccountsFromJSON() ([]APIAccount, error) {
	data, err := os.ReadFile(apiAccountsPath)
	if err != nil {
		return nil, err
	}
	var accounts []APIAccount
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func removeAccountFromJSON(refreshToken string) {
	csvMutex.Lock()
	defer csvMutex.Unlock()

	accounts, err := loadAPIAccountsFromJSON()
	if err != nil {
		return
	}
	for i, acc := range accounts {
		if acc.RefreshToken == refreshToken {
			accounts = append(accounts[:i], accounts[i+1:]...)
			if err := saveAPIAccountsToJSON(accounts); err != nil {
				NormalLogger.Printf("Failed to save accounts to JSON: %v\n", err)
			}
			return
		}
	}
}

func loadCachedAccounts() int {
	cached, err := loadAPIAccountsFromJSON()
	if err != nil || len(cached) == 0 {
		return 0
	}
	NormalLogger.Printf("Loaded %d accounts from cache\n", len(cached))
	for _, acc := range cached {
		RefreshTokens = append(RefreshTokens, Models.RefreshToken{
			ID:           acc.ID,
			Token:        acc.RefreshToken,
			ClientId:     acc.ClientID,
			ClientSecret: strings.ReplaceAll(acc.ClientSecret, "\r", ""),
		})
	}
	return len(cached)
}

func nextAvailableRefreshTokenIndexLocked() (int, bool) {
	for idx := nextRefreshTokenIndex; idx < len(RefreshTokens); idx++ {
		if RefreshTokens[idx].Disabled {
			continue
		}
		return idx, true
	}
	return 0, false
}

func consumeRefreshTokenIndexLocked(idx int) {
	if idx >= len(RefreshTokens) {
		nextRefreshTokenIndex = len(RefreshTokens)
		return
	}
	if idx < nextRefreshTokenIndex {
		return
	}
	nextRefreshTokenIndex = idx + 1
}

func banAccountViaAPI(accountID int) {
	apiURL := os.Getenv("ACCOUNT_API_URL")
	apiToken := os.Getenv("ACCOUNT_API_TOKEN")
	if apiURL == "" || apiToken == "" {
		return
	}

	reqBody, _ := json.Marshal(map[string]interface{}{"ids": []int{accountID}, "banned": true})
	req, _ := http.NewRequest("PUT", apiURL+"/update", bytes.NewBuffer(reqBody))
	req.Header.Set("X-Passkey", apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: getProxyTransport(), Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		NormalLogger.Printf("Failed to ban account %d: %v\n", accountID, err)
		return
	}
	defer resp.Body.Close()
	NormalLogger.Printf("Banned account %d via API\n", accountID)
}

func loadAccountsFromAPI(apiURL, apiToken string, count int) error {
	// Try loading from cache first
	if len(RefreshTokens) == 0 {
		loadCachedAccounts()
		if len(RefreshTokens) >= count {
			return nil
		}
	}

	categoryID := 3
	if catStr := os.Getenv("ACCOUNT_CATEGORY_ID"); catStr != "" {
		fmt.Sscanf(catStr, "%d", &categoryID)
	}

	reqBody, _ := json.Marshal(map[string]int{"category_id": categoryID, "count": count})
	req, err := http.NewRequest("POST", apiURL+"/api/accounts/fetch", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create API request: %w", err)
	}

	req.Header.Set("X-Passkey", apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: getProxyTransport(), Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch accounts from API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	var accounts []APIAccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
		return fmt.Errorf("failed to parse API response: %w", err)
	}

	for _, acc := range accounts {
		var data APIAccountData
		if err := json.Unmarshal([]byte(acc.Data), &data); err != nil {
			NormalLogger.Printf("Failed to parse account data: %v\n", err)
			continue
		}
		RefreshTokens = append(RefreshTokens, Models.RefreshToken{
			ID:           acc.ID,
			Token:        data.RefreshToken,
			ClientId:     data.ClientID,
			ClientSecret: strings.ReplaceAll(data.ClientSecret, "\r", ""),
		})
	}

	// Save all accounts to cache
	var allAccounts []APIAccount
	for _, rt := range RefreshTokens {
		if rt.Disabled {
			continue
		}
		allAccounts = append(allAccounts, APIAccount{
			ID:           rt.ID,
			RefreshToken: rt.Token,
			ClientID:     rt.ClientId,
			ClientSecret: rt.ClientSecret,
		})
	}
	if err := saveAPIAccountsToJSON(allAccounts); err != nil {
		NormalLogger.Printf("Failed to save accounts cache: %v\n", err)
	}

	if len(RefreshTokens) == 0 {
		return fmt.Errorf("no accounts received from api")
	}
	return nil
}

func DisableToken(accessToken string, reason string) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	for i, idx := range ActiveTokens {
		if RefreshTokens[idx].AccessToken.Token == accessToken {
			NormalLogger.Printf("Disabling active token %d, reason: %s\n", idx, reason)
			RefreshTokens[idx].AccessToken.ExpiresAt = 0
			RefreshTokens[idx].Disabled = true
			updateCSVEnabled(RefreshTokens[idx].Token)

			if newIdx, ok := nextAvailableRefreshTokenIndexLocked(); ok {
				var newToken Models.AccessToken
				var err error
				for attempt := 0; attempt < maxRefreshAttempt; attempt++ {
					newToken, err = GetAccessTokenFromRefreshToken(RefreshTokens[newIdx])
					if err == nil {
						break
					}
					NormalLogger.Printf("Failed to get new access token (attempt %d/%d): %v\n", attempt+1, maxRefreshAttempt, err)
				}
				if err != nil {
					ActiveTokens = append(ActiveTokens[:i], ActiveTokens[i+1:]...)
				} else {
					RefreshTokens[newIdx].AccessToken = newToken
					ActiveTokens[i] = newIdx
					consumeRefreshTokenIndexLocked(newIdx)
					NormalLogger.Printf("Rotated to new token from refresh token %d\n", newIdx)
				}
			} else {
				go fetchAndAddNewToken()
				ActiveTokens = append(ActiveTokens[:i], ActiveTokens[i+1:]...)
				NormalLogger.Printf("No more refresh tokens in pool, fetching new token\n")
			}
			break
		}
	}
}

func fetchAndAddNewToken() {
	accountSource := os.Getenv("ACCOUNT_SOURCE")
	if accountSource == "api" {
		apiURL := os.Getenv("ACCOUNT_API_URL")
		apiToken := os.Getenv("ACCOUNT_API_TOKEN")
		if apiURL == "" || apiToken == "" {
			return
		}
		if err := loadAccountsFromAPI(apiURL, apiToken, 1); err != nil {
			NormalLogger.Printf("Failed to fetch new token account from API: %v\n", err)
			return
		}
		tokenMutex.Lock()
		defer tokenMutex.Unlock()

		newIdx, ok := nextAvailableRefreshTokenIndexLocked()
		if !ok {
			return
		}

		for attempt := 0; attempt < maxRefreshAttempt; attempt++ {
			newToken, err := GetAccessTokenFromRefreshToken(RefreshTokens[newIdx])
			if err == nil {
				RefreshTokens[newIdx].AccessToken = newToken
				ActiveTokens = append(ActiveTokens, newIdx)
				consumeRefreshTokenIndexLocked(newIdx)
				NormalLogger.Printf("Added new token from API: %d\n", newIdx)
				return
			}
			NormalLogger.Printf("Failed to get new access token (attempt %d/%d): %v\n", attempt+1, maxRefreshAttempt, err)
		}
	}
}

func updateCSVEnabled(refreshToken string) {
	if csvPath != "" {
		go func() {
			csvMutex.Lock()
			defer csvMutex.Unlock()

			file, err := os.ReadFile(csvPath)
			if err != nil {
				NormalLogger.Printf("Failed to read CSV: %v\n", err)
				return
			}

			lines := strings.Split(string(file), "\n")
			for i := 1; i < len(lines); i++ {
				if strings.Contains(lines[i], refreshToken) {
					parts := strings.Split(lines[i], ",")
					if len(parts) >= 4 && strings.TrimSpace(parts[1]) == refreshToken {
						parts[0] = "False"
						lines[i] = strings.Join(parts, ",")
						break
					}
				}
			}

			_ = os.WriteFile(csvPath, []byte(strings.Join(lines, "\n")), 0644)
		}()
	} else {
		go func() {
			removeAccountFromJSON(refreshToken)
			// Find account ID and ban via API
			for _, rt := range RefreshTokens {
				if rt.Token == refreshToken && rt.ID > 0 {
					banAccountViaAPI(rt.ID)
					break
				}
			}
		}()
	}
}

func GetBearer() (string, error) {

	initDoOnce.Do(func() {
		activeTokenCountStr := os.Getenv("ACTIVE_TOKEN_COUNT")
		if activeTokenCountStr == "" {
			activeTokenCount = DefaultActiveTokenCount
		} else {
			_, _ = fmt.Sscanf(activeTokenCountStr, "%d", &activeTokenCount)
		}
		if activeTokenCount <= 0 {
			activeTokenCount = DefaultActiveTokenCount
		}

		maxRefreshAttemptStr := os.Getenv("MAX_REFRESH_ATTEMPT")
		if maxRefreshAttemptStr == "" {
			maxRefreshAttempt = DefaultMaxRefreshAttempt
		} else {
			_, _ = fmt.Sscanf(maxRefreshAttemptStr, "%d", &maxRefreshAttempt)
		}
		if maxRefreshAttempt <= 0 {
			maxRefreshAttempt = DefaultMaxRefreshAttempt
		}

		accountSource := os.Getenv("ACCOUNT_SOURCE")
		if accountSource == "" {
			accountSource = "manual"
		}

		switch accountSource {
		case "api":
			apiURL := os.Getenv("ACCOUNT_API_URL")
			apiToken := os.Getenv("ACCOUNT_API_TOKEN")
			if apiURL == "" || apiToken == "" {
				NormalLogger.Println("ACCOUNT_SOURCE=api but ACCOUNT_API_URL or ACCOUNT_API_TOKEN missing; waiting for admin configuration")
				break
			}
			NormalLogger.Printf("Loading accounts from API: %s\n", apiURL)
			if err := loadAccountsFromAPI(apiURL, apiToken, activeTokenCount); err != nil {
				NormalLogger.Printf("Failed to load accounts from API: %v\n", err)
			}
		case "csv":
			csvPath = os.Getenv("ACCOUNTS_CSV_PATH")
			if csvPath == "" {
				NormalLogger.Println("ACCOUNT_SOURCE=csv but ACCOUNTS_CSV_PATH missing; waiting for admin configuration")
				break
			}
			NormalLogger.Printf("Loading accounts from CSV: %s\n", csvPath)
			if err := loadAccountsFromCSV(csvPath); err != nil {
				NormalLogger.Printf("Failed to load accounts from CSV: %v\n", err)
			}
		default:
			loadCachedAccounts()
		}

		activateCount := activeTokenCount
		if activateCount > len(RefreshTokens) {
			activateCount = len(RefreshTokens)
		}

		tokenMutex.Lock()
		for i := 0; i < activateCount; i++ {
			var accessToken Models.AccessToken
			var err error
			for attempt := 0; attempt < maxRefreshAttempt; attempt++ {
				accessToken, err = GetAccessTokenFromRefreshToken(RefreshTokens[i])
				if err == nil {
					break
				}
				NormalLogger.Printf("Failed to get access token (attempt %d/%d): %v\n", attempt+1, maxRefreshAttempt, err)
			}
			if err != nil {
				continue
			}
			RefreshTokens[i].AccessToken = accessToken
			ActiveTokens = append(ActiveTokens, i)
			NormalLogger.Printf("Get Access Token OK for account %d\n", i)
		}
		tokenMutex.Unlock()
		nextRefreshTokenIndex = activateCount
	})

	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	now := time.Now().Unix()
	var validIndices []int

	for _, idx := range ActiveTokens {
		if RefreshTokens[idx].Disabled {
			continue
		}
		if RefreshTokens[idx].AccessToken.ExpiresAt > now {
			validIndices = append(validIndices, idx)
		}
	}

	if len(validIndices) == 0 {
		return "", fmt.Errorf("no valid access tokens available")
	}

	tokenIndex = (tokenIndex + 1) % len(validIndices)
	selectedIdx := validIndices[tokenIndex]
	return RefreshTokens[selectedIdx].AccessToken.Token, nil
}

func StartTokenRefresher() {
	go func() {
		for {
			jitter := time.Duration(rand.Intn(20)+20) * time.Minute
			time.Sleep(jitter)

			tokenMutex.Lock()
			for _, idx := range ActiveTokens {
				if RefreshTokens[idx].Disabled {
					continue
				}
				newToken, err := GetAccessTokenFromRefreshToken(RefreshTokens[idx])
				if err != nil {
					NormalLogger.Printf("Failed to refresh active token %d: %v\n", idx, err)
					continue
				}
				RefreshTokens[idx].AccessToken = newToken
				NormalLogger.Printf("Refreshed active token %d\n", idx)
			}
			tokenMutex.Unlock()
		}
	}()
}

func GetAccessTokenFromRefreshToken(refreshToken Models.RefreshToken) (Models.AccessToken, error) {
	// Prepare request body
	requestBody := Models.TokenRefreshRequest{
		ClientId:     refreshToken.ClientId,
		ClientSecret: refreshToken.ClientSecret,
		GrantType:    "refresh_token",
		RefreshToken: refreshToken.Token,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return Models.AccessToken{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	qUrl := os.Getenv("OIDC_URL")
	if strings.TrimSpace(qUrl) == "" {
		return Models.AccessToken{}, fmt.Errorf("OIDC_URL not configured; please set it via environment variable or admin panel")
	}
	req, err := http.NewRequest("POST", qUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return Models.AccessToken{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to match the curl command
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("user-agent", "aws-sdk-rust/1.3.10 os/macos lang/rust/1.88.0")
	req.Header.Set("x-amz-user-agent", "aws-sdk-rust/1.3.10 ua/2.1 api/ssooidc/1.89.0 os/macos lang/rust/1.88.0 m/E app/AmazonQ-For-KIRO_CLI")
	req.Header.Set("amz-sdk-request", "attempt=1; max=3")
	req.Header.Set("amz-sdk-invocation-id", uuid.New().String())
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "gzip")

	// Create HTTP client and make request
	client := &http.Client{
		Transport: getProxyTransport(),
		Timeout:   30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return Models.AccessToken{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Models.AccessToken{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return Models.AccessToken{}, fmt.Errorf("token refresh failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResponse Models.TokenRefreshResponse
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return Models.AccessToken{}, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Unix() + int64(tokenResponse.ExpiresIn)

	return Models.AccessToken{
		Token:     tokenResponse.AccessToken,
		ExpiresAt: expiresAt,
	}, nil
}

func CheckAndDisableToken(body []byte, token string) {
	bodyStr := string(body)
	if strings.Contains(bodyStr, "MONTHLY_REQUEST_COUNT") {
		DisableToken(token, "MONTHLY_REQUEST_COUNT")
	} else if strings.Contains(bodyStr, "TEMPORARILY_SUSPENDED") {
		DisableToken(token, "TEMPORARILY_SUSPENDED")
	}
}
