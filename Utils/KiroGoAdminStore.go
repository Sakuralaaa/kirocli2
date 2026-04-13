package Utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type KiroGoAccountMeta struct {
	ID                int     `json:"id"`
	Email             string  `json:"email,omitempty"`
	UserID            string  `json:"userId,omitempty"`
	Nickname          string  `json:"nickname,omitempty"`
	AuthMethod        string  `json:"authMethod,omitempty"`
	Provider          string  `json:"provider,omitempty"`
	Region            string  `json:"region,omitempty"`
	Enabled           bool    `json:"enabled"`
	BanStatus         string  `json:"banStatus,omitempty"`
	BanReason         string  `json:"banReason,omitempty"`
	BanTime           int64   `json:"banTime,omitempty"`
	MachineID         string  `json:"machineId,omitempty"`
	Weight            int     `json:"weight,omitempty"`
	SubscriptionType  string  `json:"subscriptionType,omitempty"`
	SubscriptionTitle string  `json:"subscriptionTitle,omitempty"`
	DaysRemaining     int     `json:"daysRemaining,omitempty"`
	UsageCurrent      float64 `json:"usageCurrent,omitempty"`
	UsageLimit        float64 `json:"usageLimit,omitempty"`
	UsagePercent      float64 `json:"usagePercent,omitempty"`
	NextResetDate     string  `json:"nextResetDate,omitempty"`
	LastRefresh       int64   `json:"lastRefresh,omitempty"`
	TrialUsageCurrent float64 `json:"trialUsageCurrent,omitempty"`
	TrialUsageLimit   float64 `json:"trialUsageLimit,omitempty"`
	TrialUsagePercent float64 `json:"trialUsagePercent,omitempty"`
	TrialStatus       string  `json:"trialStatus,omitempty"`
	TrialExpiresAt    int64   `json:"trialExpiresAt,omitempty"`
	RequestCount      int     `json:"requestCount,omitempty"`
	ErrorCount        int     `json:"errorCount,omitempty"`
	TotalTokens       int     `json:"totalTokens,omitempty"`
	TotalCredits      float64 `json:"totalCredits,omitempty"`
	LastUsed          int64   `json:"lastUsed,omitempty"`
}

type KiroGoAdminStore struct {
	Password             string              `json:"password"`
	APIKey               string              `json:"apiKey,omitempty"`
	RequireAPIKey        bool                `json:"requireApiKey"`
	ThinkingSuffix       string              `json:"thinkingSuffix,omitempty"`
	OpenAIThinkingFormat string              `json:"openaiThinkingFormat,omitempty"`
	ClaudeThinkingFormat string              `json:"claudeThinkingFormat,omitempty"`
	PreferredEndpoint    string              `json:"preferredEndpoint,omitempty"`
	TotalRequests        int64               `json:"totalRequests,omitempty"`
	SuccessRequests      int64               `json:"successRequests,omitempty"`
	FailedRequests       int64               `json:"failedRequests,omitempty"`
	TotalTokens          int64               `json:"totalTokens,omitempty"`
	TotalCredits         float64             `json:"totalCredits,omitempty"`
	StartTime            int64               `json:"startTime,omitempty"`
	Accounts             []KiroGoAccountMeta `json:"accounts"`
}

type KiroGoThinkingConfig struct {
	Suffix       string `json:"suffix"`
	OpenAIFormat string `json:"openaiFormat"`
	ClaudeFormat string `json:"claudeFormat"`
}

var (
	kiroGoAdminStoreMu   sync.RWMutex
	kiroGoAdminStoreOnce sync.Once
	kiroGoAdminStore     KiroGoAdminStore
)

func kiroGoAdminStorePath() string {
	return "data/kirogo_admin_store.json"
}

func defaultKiroGoAdminStore() KiroGoAdminStore {
	pwd := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
	if pwd == "" {
		pwd = strings.TrimSpace(os.Getenv("BEARER_TOKEN"))
	}
	if pwd == "" {
		pwd = uuid.NewString()
		_ = os.Setenv("ADMIN_TOKEN", pwd)
		if NormalLogger != nil {
			NormalLogger.Printf("Kiro-Go admin bootstrap password generated: %s\n", pwd)
		}
	}
	return KiroGoAdminStore{
		Password:             pwd,
		RequireAPIKey:        false,
		ThinkingSuffix:       "-thinking",
		OpenAIThinkingFormat: "reasoning_content",
		ClaudeThinkingFormat: "thinking",
		PreferredEndpoint:    "auto",
		StartTime:            time.Now().Unix(),
		Accounts:             []KiroGoAccountMeta{},
	}
}

func ensureKiroGoAdminStoreLoaded() {
	kiroGoAdminStoreOnce.Do(func() {
		store := defaultKiroGoAdminStore()
		path := kiroGoAdminStorePath()
		data, err := os.ReadFile(path)
		if err == nil {
			_ = json.Unmarshal(data, &store)
		}
		if store.StartTime <= 0 {
			store.StartTime = time.Now().Unix()
		}
		if store.ThinkingSuffix == "" {
			store.ThinkingSuffix = "-thinking"
		}
		if store.OpenAIThinkingFormat == "" {
			store.OpenAIThinkingFormat = "reasoning_content"
		}
		if store.ClaudeThinkingFormat == "" {
			store.ClaudeThinkingFormat = "thinking"
		}
		if store.PreferredEndpoint == "" {
			store.PreferredEndpoint = "auto"
		}
		kiroGoAdminStore = store
		_ = persistKiroGoAdminStoreLocked()
	})
}

func persistKiroGoAdminStoreLocked() error {
	path := kiroGoAdminStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(kiroGoAdminStore, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func GetKiroGoAdminStore() KiroGoAdminStore {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.RLock()
	defer kiroGoAdminStoreMu.RUnlock()
	return kiroGoAdminStore
}

func GetKiroGoAdminPassword() string {
	return GetKiroGoAdminStore().Password
}

func UpdateKiroGoAdminSettings(apiKey *string, requireAPIKey *bool, password *string) error {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.Lock()
	defer kiroGoAdminStoreMu.Unlock()
	if apiKey != nil {
		kiroGoAdminStore.APIKey = strings.TrimSpace(*apiKey)
	}
	if requireAPIKey != nil {
		kiroGoAdminStore.RequireAPIKey = *requireAPIKey
	}
	if password != nil && strings.TrimSpace(*password) != "" {
		kiroGoAdminStore.Password = strings.TrimSpace(*password)
		_ = os.Setenv("ADMIN_TOKEN", kiroGoAdminStore.Password)
	}
	return persistKiroGoAdminStoreLocked()
}

func UpdateKiroGoThinkingConfig(suffix, openaiFormat, claudeFormat string) error {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.Lock()
	defer kiroGoAdminStoreMu.Unlock()
	if strings.TrimSpace(suffix) != "" {
		kiroGoAdminStore.ThinkingSuffix = strings.TrimSpace(suffix)
	}
	if strings.TrimSpace(openaiFormat) != "" {
		kiroGoAdminStore.OpenAIThinkingFormat = strings.TrimSpace(openaiFormat)
	}
	if strings.TrimSpace(claudeFormat) != "" {
		kiroGoAdminStore.ClaudeThinkingFormat = strings.TrimSpace(claudeFormat)
	}
	return persistKiroGoAdminStoreLocked()
}

func GetKiroGoThinkingConfig() KiroGoThinkingConfig {
	store := GetKiroGoAdminStore()
	return KiroGoThinkingConfig{
		Suffix:       store.ThinkingSuffix,
		OpenAIFormat: store.OpenAIThinkingFormat,
		ClaudeFormat: store.ClaudeThinkingFormat,
	}
}

func UpdateKiroGoPreferredEndpoint(endpoint string) error {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.Lock()
	defer kiroGoAdminStoreMu.Unlock()
	kiroGoAdminStore.PreferredEndpoint = strings.TrimSpace(endpoint)
	if kiroGoAdminStore.PreferredEndpoint == "" {
		kiroGoAdminStore.PreferredEndpoint = "auto"
	}
	return persistKiroGoAdminStoreLocked()
}

func GetKiroGoPreferredEndpoint() string {
	store := GetKiroGoAdminStore()
	if strings.TrimSpace(store.PreferredEndpoint) == "" {
		return "auto"
	}
	return store.PreferredEndpoint
}

func UpsertKiroGoAccountMeta(meta KiroGoAccountMeta) error {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.Lock()
	defer kiroGoAdminStoreMu.Unlock()
	for i := range kiroGoAdminStore.Accounts {
		if kiroGoAdminStore.Accounts[i].ID == meta.ID {
			if meta.Email != "" {
				kiroGoAdminStore.Accounts[i].Email = meta.Email
			}
			if meta.UserID != "" {
				kiroGoAdminStore.Accounts[i].UserID = meta.UserID
			}
			if meta.Nickname != "" {
				kiroGoAdminStore.Accounts[i].Nickname = meta.Nickname
			}
			if meta.AuthMethod != "" {
				kiroGoAdminStore.Accounts[i].AuthMethod = meta.AuthMethod
			}
			if meta.Provider != "" {
				kiroGoAdminStore.Accounts[i].Provider = meta.Provider
			}
			if meta.Region != "" {
				kiroGoAdminStore.Accounts[i].Region = meta.Region
			}
			if meta.MachineID != "" {
				kiroGoAdminStore.Accounts[i].MachineID = meta.MachineID
			}
			if meta.BanStatus != "" {
				kiroGoAdminStore.Accounts[i].BanStatus = meta.BanStatus
			}
			kiroGoAdminStore.Accounts[i].Enabled = meta.Enabled
			kiroGoAdminStore.Accounts[i].Weight = meta.Weight
			return persistKiroGoAdminStoreLocked()
		}
	}
	kiroGoAdminStore.Accounts = append(kiroGoAdminStore.Accounts, meta)
	return persistKiroGoAdminStoreLocked()
}

func DeleteKiroGoAccountMeta(id int) error {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.Lock()
	defer kiroGoAdminStoreMu.Unlock()
	next := make([]KiroGoAccountMeta, 0, len(kiroGoAdminStore.Accounts))
	for _, item := range kiroGoAdminStore.Accounts {
		if item.ID != id {
			next = append(next, item)
		}
	}
	kiroGoAdminStore.Accounts = next
	return persistKiroGoAdminStoreLocked()
}

func GetKiroGoAccountMeta(id int) (KiroGoAccountMeta, bool) {
	store := GetKiroGoAdminStore()
	for _, item := range store.Accounts {
		if item.ID == id {
			return item, true
		}
	}
	return KiroGoAccountMeta{}, false
}

func ListKiroGoAccountMetas() []KiroGoAccountMeta {
	store := GetKiroGoAdminStore()
	return store.Accounts
}

func KiroGoStatsSnapshot() map[string]interface{} {
	store := GetKiroGoAdminStore()
	return map[string]interface{}{
		"totalRequests":   store.TotalRequests,
		"successRequests": store.SuccessRequests,
		"failedRequests":  store.FailedRequests,
		"totalTokens":     store.TotalTokens,
		"totalCredits":    store.TotalCredits,
		"uptime":          time.Now().Unix() - store.StartTime,
	}
}

func ResetKiroGoStats() error {
	ensureKiroGoAdminStoreLoaded()
	kiroGoAdminStoreMu.Lock()
	defer kiroGoAdminStoreMu.Unlock()
	kiroGoAdminStore.TotalRequests = 0
	kiroGoAdminStore.SuccessRequests = 0
	kiroGoAdminStore.FailedRequests = 0
	kiroGoAdminStore.TotalTokens = 0
	kiroGoAdminStore.TotalCredits = 0
	return persistKiroGoAdminStoreLocked()
}

func ParseKiroGoID(raw string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(raw))
}
