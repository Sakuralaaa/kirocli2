package Utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const defaultRuntimeConfigPath = "data/config.json"

type RuntimeConfig struct {
	Port              string `json:"port"`
	GinMode           string `json:"gin_mode"`
	BearerToken       string `json:"bearer_token"`
	AdminToken        string `json:"admin_token"`
	OIDCURL           string `json:"oidc_url"`
	AmazonQURL        string `json:"amazon_q_url"`
	ProxyURL          string `json:"proxy_url"`
	AccountSource     string `json:"account_source"`
	AccountsCSVPath   string `json:"accounts_csv_path"`
	AccountAPIURL     string `json:"account_api_url"`
	AccountAPIToken   string `json:"account_api_token"`
	AccountCategoryID string `json:"account_category_id"`
	ActiveTokenCount  string `json:"active_token_count"`
	MaxRefreshAttempt string `json:"max_refresh_attempt"`
}

func runtimeConfigPath() string {
	path := strings.TrimSpace(os.Getenv("CONFIG_PATH"))
	if path == "" {
		return defaultRuntimeConfigPath
	}
	return path
}

func defaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Port:              "4000",
		GinMode:           "release",
		AccountSource:     "manual",
		AccountCategoryID: "3",
		ActiveTokenCount:  "10",
		MaxRefreshAttempt: "3",
	}
}

func loadRuntimeConfigFile(path string) (RuntimeConfig, error) {
	cfg := defaultRuntimeConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func saveRuntimeConfigFile(path string, cfg RuntimeConfig) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func setEnvIfEmpty(key, value string) error {
	if strings.TrimSpace(os.Getenv(key)) != "" || strings.TrimSpace(value) == "" {
		return nil
	}
	return os.Setenv(key, strings.TrimSpace(value))
}

func applyConfigToEnv(cfg RuntimeConfig) error {
	pairs := map[string]string{
		"PORT":                cfg.Port,
		"GIN_MODE":            cfg.GinMode,
		"BEARER_TOKEN":        cfg.BearerToken,
		"ADMIN_TOKEN":         cfg.AdminToken,
		"OIDC_URL":            cfg.OIDCURL,
		"AMAZON_Q_URL":        cfg.AmazonQURL,
		"PROXY_URL":           cfg.ProxyURL,
		"ACCOUNT_SOURCE":      cfg.AccountSource,
		"ACCOUNTS_CSV_PATH":   cfg.AccountsCSVPath,
		"ACCOUNT_API_URL":     cfg.AccountAPIURL,
		"ACCOUNT_API_TOKEN":   cfg.AccountAPIToken,
		"ACCOUNT_CATEGORY_ID": cfg.AccountCategoryID,
		"ACTIVE_TOKEN_COUNT":  cfg.ActiveTokenCount,
		"MAX_REFRESH_ATTEMPT": cfg.MaxRefreshAttempt,
	}
	for key, value := range pairs {
		if err := setEnvIfEmpty(key, value); err != nil {
			return err
		}
	}
	return nil
}

func InitRuntimeConfig() error {
	path := runtimeConfigPath()
	cfg, err := loadRuntimeConfigFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = defaultRuntimeConfig()
			if writeErr := saveRuntimeConfigFile(path, cfg); writeErr != nil {
				return writeErr
			}
		} else {
			return err
		}
	}
	return applyConfigToEnv(cfg)
}

func SaveRuntimeConfigFromEnv() error {
	path := runtimeConfigPath()
	cfg := RuntimeConfig{
		Port:              strings.TrimSpace(os.Getenv("PORT")),
		GinMode:           strings.TrimSpace(os.Getenv("GIN_MODE")),
		BearerToken:       strings.TrimSpace(os.Getenv("BEARER_TOKEN")),
		AdminToken:        strings.TrimSpace(os.Getenv("ADMIN_TOKEN")),
		OIDCURL:           strings.TrimSpace(os.Getenv("OIDC_URL")),
		AmazonQURL:        strings.TrimSpace(os.Getenv("AMAZON_Q_URL")),
		ProxyURL:          strings.TrimSpace(os.Getenv("PROXY_URL")),
		AccountSource:     strings.TrimSpace(os.Getenv("ACCOUNT_SOURCE")),
		AccountsCSVPath:   strings.TrimSpace(os.Getenv("ACCOUNTS_CSV_PATH")),
		AccountAPIURL:     strings.TrimSpace(os.Getenv("ACCOUNT_API_URL")),
		AccountAPIToken:   strings.TrimSpace(os.Getenv("ACCOUNT_API_TOKEN")),
		AccountCategoryID: strings.TrimSpace(os.Getenv("ACCOUNT_CATEGORY_ID")),
		ActiveTokenCount:  strings.TrimSpace(os.Getenv("ACTIVE_TOKEN_COUNT")),
		MaxRefreshAttempt: strings.TrimSpace(os.Getenv("MAX_REFRESH_ATTEMPT")),
	}

	defaults := defaultRuntimeConfig()
	if cfg.Port == "" {
		cfg.Port = defaults.Port
	}
	if cfg.GinMode == "" {
		cfg.GinMode = defaults.GinMode
	}
	if cfg.AccountSource == "" {
		cfg.AccountSource = defaults.AccountSource
	}
	if cfg.AccountCategoryID == "" {
		cfg.AccountCategoryID = defaults.AccountCategoryID
	}
	if cfg.ActiveTokenCount == "" {
		cfg.ActiveTokenCount = defaults.ActiveTokenCount
	}
	if cfg.MaxRefreshAttempt == "" {
		cfg.MaxRefreshAttempt = defaults.MaxRefreshAttempt
	}

	return saveRuntimeConfigFile(path, cfg)
}
