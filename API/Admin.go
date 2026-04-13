package API

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kilocli2api/KiroAuth"
	"kilocli2api/Utils"
)

const kiroGoVersion = "1.0.3"

type kiroGoAccountRequest struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	UserID       string `json:"userId"`
	Nickname     string `json:"nickname"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	AuthMethod   string `json:"authMethod"`
	Provider     string `json:"provider"`
	Region       string `json:"region"`
	ExpiresAt    int64  `json:"expiresAt"`
	MachineID    string `json:"machineId"`
	Enabled      *bool  `json:"enabled"`
	Weight       int    `json:"weight"`
}

func AdminPanel(c *gin.Context) {
	c.File("web/index.html")
}

func AdminStatic(c *gin.Context) {
	path := strings.TrimPrefix(c.Param("filepath"), "/")
	if path == "" {
		c.File("web/index.html")
		return
	}
	c.File("web/" + path)
}

func AdminStatus(c *gin.Context) {
	snapshot := Utils.GetAdminSnapshot()
	store := Utils.GetKiroGoAdminStore()
	stats := Utils.KiroGoStatsSnapshot()

	c.JSON(http.StatusOK, gin.H{
		"accounts":        snapshot.TotalAccounts,
		"available":       snapshot.ActiveAccountCount,
		"totalRequests":   stats["totalRequests"],
		"successRequests": stats["successRequests"],
		"failedRequests":  stats["failedRequests"],
		"totalTokens":     stats["totalTokens"],
		"totalCredits":    stats["totalCredits"],
		"uptime":          stats["uptime"],
		"requireApiKey":   store.RequireAPIKey,
	})
}

func accountIDFromPath(c *gin.Context) (int, error) {
	id, err := Utils.ParseAccountID(c.Param("id"))
	if err == nil {
		return id, nil
	}
	return 0, errors.New("invalid account id")
}

func accountMetaFromRequest(id int, req kiroGoAccountRequest) Utils.KiroGoAccountMeta {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	region := req.Region
	if strings.TrimSpace(region) == "" {
		region = "us-east-1"
	}
	machineID := strings.TrimSpace(req.MachineID)
	if machineID == "" {
		machineID = uuid.NewString()
	}
	authMethod := strings.TrimSpace(req.AuthMethod)
	if authMethod == "" {
		if strings.TrimSpace(req.ClientID) != "" {
			authMethod = "idc"
		} else {
			authMethod = "social"
		}
	}

	return Utils.KiroGoAccountMeta{
		ID:         id,
		Email:      strings.TrimSpace(req.Email),
		UserID:     strings.TrimSpace(req.UserID),
		Nickname:   strings.TrimSpace(req.Nickname),
		AuthMethod: authMethod,
		Provider:   strings.TrimSpace(req.Provider),
		Region:     region,
		Enabled:    enabled,
		MachineID:  machineID,
		Weight:     req.Weight,
	}
}

func addOrImportAccount(req kiroGoAccountRequest) (int, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	refreshToken := strings.TrimSpace(req.RefreshToken)
	clientID := strings.TrimSpace(req.ClientID)
	clientSecret := strings.TrimSpace(req.ClientSecret)
	if refreshToken == "" {
		return 0, errors.New("refreshToken is required")
	}

	added, err := Utils.AddImportedAccountWithAccessToken(refreshToken, clientID, clientSecret, strings.TrimSpace(req.AccessToken), req.ExpiresAt, enabled)
	if err != nil {
		return 0, err
	}

	meta := accountMetaFromRequest(added.ID, req)
	if err := Utils.UpsertKiroGoAccountMeta(meta); err != nil {
		return 0, err
	}
	return added.ID, nil
}

func AdminGetAccounts(c *gin.Context) {
	snapshots := Utils.ListAdminAccounts()
	metas := Utils.ListKiroGoAccountMetas()
	metaMap := map[int]Utils.KiroGoAccountMeta{}
	for _, m := range metas {
		metaMap[m.ID] = m
	}

	result := make([]gin.H, 0, len(snapshots))
	for _, s := range snapshots {
		m, ok := metaMap[s.ID]
		if !ok {
			m = Utils.KiroGoAccountMeta{ID: s.ID, Enabled: !s.Disabled, Region: "us-east-1", AuthMethod: "idc", MachineID: uuid.NewString()}
			_ = Utils.UpsertKiroGoAccountMeta(m)
		}
		result = append(result, gin.H{
			"id":                strconv.Itoa(s.ID),
			"email":             m.Email,
			"userId":            m.UserID,
			"nickname":          m.Nickname,
			"authMethod":        m.AuthMethod,
			"provider":          m.Provider,
			"region":            m.Region,
			"enabled":           !s.Disabled,
			"banStatus":         m.BanStatus,
			"banReason":         m.BanReason,
			"banTime":           m.BanTime,
			"expiresAt":         s.ExpiresAt,
			"hasToken":          s.ExpiresAt > time.Now().Unix(),
			"machineId":         m.MachineID,
			"weight":            m.Weight,
			"subscriptionType":  m.SubscriptionType,
			"subscriptionTitle": m.SubscriptionTitle,
			"daysRemaining":     m.DaysRemaining,
			"usageCurrent":      m.UsageCurrent,
			"usageLimit":        m.UsageLimit,
			"usagePercent":      m.UsagePercent,
			"nextResetDate":     m.NextResetDate,
			"lastRefresh":       m.LastRefresh,
			"trialUsageCurrent": m.TrialUsageCurrent,
			"trialUsageLimit":   m.TrialUsageLimit,
			"trialUsagePercent": m.TrialUsagePercent,
			"trialStatus":       m.TrialStatus,
			"trialExpiresAt":    m.TrialExpiresAt,
			"requestCount":      m.RequestCount,
			"errorCount":        m.ErrorCount,
			"totalTokens":       m.TotalTokens,
			"totalCredits":      m.TotalCredits,
			"lastUsed":          m.LastUsed,
		})
	}
	c.JSON(http.StatusOK, result)
}

func AdminAddAccount(c *gin.Context) {
	var req kiroGoAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	id, err := addOrImportAccount(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "id": strconv.Itoa(id)})
}

func AdminDeleteAccount(c *gin.Context) {
	id, err := accountIDFromPath(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Utils.DeleteAccountByID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = Utils.DeleteKiroGoAccountMeta(id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminUpdateAccount(c *gin.Context) {
	id, err := accountIDFromPath(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	var enabled *bool
	if v, ok := req["enabled"].(bool); ok {
		enabled = &v
	}

	_, updErr := Utils.UpdateAccountByID(id, enabled, nil, nil, nil)
	if updErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": updErr.Error()})
		return
	}

	meta, ok := Utils.GetKiroGoAccountMeta(id)
	if !ok {
		meta = Utils.KiroGoAccountMeta{ID: id, Enabled: enabled == nil || *enabled, Region: "us-east-1", AuthMethod: "idc", MachineID: uuid.NewString()}
	}
	if enabled != nil {
		meta.Enabled = *enabled
	}
	if v, ok := req["nickname"].(string); ok {
		meta.Nickname = strings.TrimSpace(v)
	}
	if v, ok := req["machineId"].(string); ok {
		meta.MachineID = strings.TrimSpace(v)
	}
	if v, ok := req["weight"].(float64); ok {
		meta.Weight = int(v)
	}

	_ = Utils.UpsertKiroGoAccountMeta(meta)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminBatchAccounts(c *gin.Context) {
	var req struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No account IDs provided"})
		return
	}

	switch req.Action {
	case "enable", "disable":
		target := req.Action == "enable"
		count := 0
		for _, raw := range req.IDs {
			id, err := Utils.ParseAccountID(raw)
			if err != nil {
				continue
			}
			if _, err := Utils.UpdateAccountByID(id, &target, nil, nil, nil); err == nil {
				meta, ok := Utils.GetKiroGoAccountMeta(id)
				if !ok {
					meta = Utils.KiroGoAccountMeta{ID: id, Enabled: target, Region: "us-east-1", AuthMethod: "idc", MachineID: uuid.NewString()}
				}
				meta.Enabled = target
				_ = Utils.UpsertKiroGoAccountMeta(meta)
				count++
			}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "count": count})
	case "refresh":
		refreshed := 0
		failed := 0
		for _, raw := range req.IDs {
			id, err := Utils.ParseAccountID(raw)
			if err != nil {
				failed++
				continue
			}
			account, _, ok := Utils.GetAccountTokenStateByID(id)
			if !ok {
				failed++
				continue
			}
			token, err := Utils.GetAccessTokenFromRefreshToken(account)
			if err != nil {
				failed++
				continue
			}
			_ = Utils.SetAccountTokenByID(id, token.Token, token.ExpiresAt)
			refreshed++
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "refreshed": refreshed, "failed": failed})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action: " + req.Action})
	}
}

func AdminRefreshAccount(c *gin.Context) {
	id, err := accountIDFromPath(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, _, ok := Utils.GetAccountTokenStateByID(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	token, err := Utils.GetAccessTokenFromRefreshToken(account)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = Utils.SetAccountTokenByID(id, token.Token, token.ExpiresAt)
	c.JSON(http.StatusOK, gin.H{"success": true, "info": gin.H{"expiresAt": token.ExpiresAt}})
}

func AdminGetAccountFull(c *gin.Context) {
	id, err := accountIDFromPath(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, active, ok := Utils.GetAccountTokenStateByID(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	meta, _ := Utils.GetKiroGoAccountMeta(id)
	c.JSON(http.StatusOK, gin.H{
		"id":           strconv.Itoa(id),
		"email":        meta.Email,
		"userId":       meta.UserID,
		"nickname":     meta.Nickname,
		"accessToken":  account.AccessToken.Token,
		"refreshToken": account.Token,
		"clientId":     account.ClientId,
		"clientSecret": account.ClientSecret,
		"authMethod":   meta.AuthMethod,
		"provider":     meta.Provider,
		"region":       meta.Region,
		"expiresAt":    account.AccessToken.ExpiresAt,
		"machineId":    meta.MachineID,
		"enabled":      !account.Disabled,
		"active":       active,
		"weight":       meta.Weight,
	})
}

func AdminGetAccountModels(c *gin.Context) {
	qModels, err := fetchQModels()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	models := make([]gin.H, 0, len(qModels.Models))
	for _, item := range qModels.Models {
		models = append(models, gin.H{"id": item.ModelID, "name": item.ModelName})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "models": models})
}

func AdminGenerateMachineID(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"machineId": uuid.NewString()})
}

func AdminGetSettings(c *gin.Context) {
	store := Utils.GetKiroGoAdminStore()
	c.JSON(http.StatusOK, gin.H{
		"apiKey":        store.APIKey,
		"requireApiKey": store.RequireAPIKey,
		"port":          os.Getenv("PORT"),
		"host":          "0.0.0.0",
	})
}

func AdminUpdateSettings(c *gin.Context) {
	var req struct {
		APIKey        *string `json:"apiKey"`
		RequireAPIKey *bool   `json:"requireApiKey"`
		Password      *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if err := Utils.UpdateKiroGoAdminSettings(req.APIKey, req.RequireAPIKey, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.Password != nil {
		_ = Utils.SaveRuntimeConfigFromEnv()
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminGetStats(c *gin.Context) {
	c.JSON(http.StatusOK, Utils.KiroGoStatsSnapshot())
}

func AdminResetStats(c *gin.Context) {
	if err := Utils.ResetKiroGoStats(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminGetThinkingConfig(c *gin.Context) {
	cfg := Utils.GetKiroGoThinkingConfig()
	c.JSON(http.StatusOK, gin.H{"suffix": cfg.Suffix, "openaiFormat": cfg.OpenAIFormat, "claudeFormat": cfg.ClaudeFormat})
}

func AdminUpdateThinkingConfig(c *gin.Context) {
	var req struct {
		Suffix       string `json:"suffix"`
		OpenAIFormat string `json:"openaiFormat"`
		ClaudeFormat string `json:"claudeFormat"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	valid := map[string]bool{"reasoning_content": true, "thinking": true, "think": true}
	if req.OpenAIFormat != "" && !valid[req.OpenAIFormat] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid openaiFormat, must be: reasoning_content, thinking, or think"})
		return
	}
	if req.ClaudeFormat != "" && !valid[req.ClaudeFormat] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid claudeFormat, must be: reasoning_content, thinking, or think"})
		return
	}
	if err := Utils.UpdateKiroGoThinkingConfig(req.Suffix, req.OpenAIFormat, req.ClaudeFormat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminGetEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"preferredEndpoint": Utils.GetKiroGoPreferredEndpoint()})
}

func AdminUpdateEndpoint(c *gin.Context) {
	var req struct {
		PreferredEndpoint string `json:"preferredEndpoint"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	valid := map[string]bool{"auto": true, "codewhisperer": true, "amazonq": true}
	if !valid[req.PreferredEndpoint] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid endpoint, must be: auto, codewhisperer, or amazonq"})
		return
	}
	if err := Utils.UpdateKiroGoPreferredEndpoint(req.PreferredEndpoint); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": kiroGoVersion})
}

func AdminExportAccounts(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	_ = c.ShouldBindJSON(&req)

	snapshots := Utils.ListAdminAccounts()
	selected := map[string]bool{}
	for _, id := range req.IDs {
		selected[id] = true
	}

	type ExportCredentials struct {
		AccessToken  string `json:"accessToken"`
		CsrfToken    string `json:"csrfToken"`
		RefreshToken string `json:"refreshToken"`
		ClientID     string `json:"clientId,omitempty"`
		ClientSecret string `json:"clientSecret,omitempty"`
		Region       string `json:"region,omitempty"`
		ExpiresAt    int64  `json:"expiresAt"`
		AuthMethod   string `json:"authMethod,omitempty"`
		Provider     string `json:"provider,omitempty"`
	}
	type ExportAccount struct {
		ID          string            `json:"id"`
		Email       string            `json:"email"`
		Nickname    string            `json:"nickname,omitempty"`
		Idp         string            `json:"idp"`
		UserId      string            `json:"userId,omitempty"`
		MachineId   string            `json:"machineId,omitempty"`
		Credentials ExportCredentials `json:"credentials"`
		Tags        []string          `json:"tags"`
		Status      string            `json:"status"`
		CreatedAt   int64             `json:"createdAt"`
		LastUsedAt  int64             `json:"lastUsedAt"`
	}

	out := make([]ExportAccount, 0, len(snapshots))
	for _, s := range snapshots {
		sid := strconv.Itoa(s.ID)
		if len(selected) > 0 && !selected[sid] {
			continue
		}
		account, _, ok := Utils.GetAccountTokenStateByID(s.ID)
		if !ok {
			continue
		}
		meta, _ := Utils.GetKiroGoAccountMeta(s.ID)
		idp := meta.Provider
		if idp == "" {
			if strings.EqualFold(meta.AuthMethod, "social") {
				idp = "Google"
			} else {
				idp = "BuilderId"
			}
		}

		out = append(out, ExportAccount{
			ID:        sid,
			Email:     meta.Email,
			Nickname:  meta.Nickname,
			Idp:       idp,
			UserId:    meta.UserID,
			MachineId: meta.MachineID,
			Credentials: ExportCredentials{
				AccessToken:  account.AccessToken.Token,
				CsrfToken:    "",
				RefreshToken: account.Token,
				ClientID:     account.ClientId,
				ClientSecret: account.ClientSecret,
				Region:       meta.Region,
				ExpiresAt:    account.AccessToken.ExpiresAt * 1000,
				AuthMethod:   meta.AuthMethod,
				Provider:     meta.Provider,
			},
			Tags:       []string{},
			Status:     "active",
			CreatedAt:  time.Now().UnixMilli(),
			LastUsedAt: time.Now().UnixMilli(),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"version":    kiroGoVersion,
		"exportedAt": time.Now().UnixMilli(),
		"accounts":   out,
		"groups":     []interface{}{},
		"tags":       []interface{}{},
	})
}

func AdminStartBuilderID(c *gin.Context) {
	var req struct {
		Region string `json:"region"`
	}
	_ = c.ShouldBindJSON(&req)
	session, err := KiroAuth.StartBuilderIdLogin(req.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"sessionId":       session.ID,
		"userCode":        session.UserCode,
		"verificationUri": session.VerificationUri,
		"interval":        session.Interval,
	})
}

func AdminPollBuilderID(c *gin.Context) {
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	accessToken, refreshToken, clientID, clientSecret, region, expiresIn, status, err := KiroAuth.PollBuilderIdAuth(req.SessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if status == "pending" || status == "slow_down" {
		interval := 5
		if session := KiroAuth.GetBuilderIdSession(req.SessionID); session != nil {
			interval = session.Interval
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "completed": false, "status": status, "interval": interval})
		return
	}

	reqData := kiroGoAccountRequest{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthMethod:   "idc",
		Provider:     "BuilderId",
		Region:       region,
		Enabled:      boolPtr(true),
		ExpiresAt:    time.Now().Unix() + int64(expiresIn),
	}
	email, userID, _ := KiroAuth.GetUserInfo(accessToken)
	reqData.Email = email
	reqData.UserID = userID
	reqData.MachineID = uuid.NewString()

	id, addErr := addOrImportAccount(reqData)
	if addErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": addErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "completed": true, "account": gin.H{"id": strconv.Itoa(id), "email": email}})
}

func boolPtr(v bool) *bool { return &v }

func AdminStartIamSSO(c *gin.Context) {
	var req struct {
		StartURL string `json:"startUrl"`
		Region   string `json:"region"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if strings.TrimSpace(req.StartURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "startUrl is required"})
		return
	}

	sessionID, authorizeURL, expiresIn, err := KiroAuth.StartIamSsoLogin(req.StartURL, req.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sessionId": sessionID, "authorizeUrl": authorizeURL, "expiresIn": expiresIn})
}

func AdminCompleteIamSSO(c *gin.Context) {
	var req struct {
		SessionID   string `json:"sessionId"`
		CallbackURL string `json:"callbackUrl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	accessToken, refreshToken, clientID, clientSecret, region, expiresIn, err := KiroAuth.CompleteIamSsoLogin(req.SessionID, req.CallbackURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	email, userID, _ := KiroAuth.GetUserInfo(accessToken)
	reqData := kiroGoAccountRequest{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthMethod:   "idc",
		Region:       region,
		Enabled:      boolPtr(true),
		ExpiresAt:    time.Now().Unix() + int64(expiresIn),
		Email:        email,
		UserID:       userID,
		MachineID:    uuid.NewString(),
	}

	id, addErr := addOrImportAccount(reqData)
	if addErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": addErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "account": gin.H{"id": strconv.Itoa(id), "email": email}})
}

func AdminImportSsoToken(c *gin.Context) {
	var req struct {
		BearerToken string `json:"bearerToken"`
		Region      string `json:"region"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	if strings.TrimSpace(req.BearerToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bearerToken is required"})
		return
	}

	tokens := strings.Split(strings.TrimSpace(req.BearerToken), "\n")
	imported := make([]gin.H, 0)
	errs := make([]string, 0)

	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		accessToken, refreshToken, clientID, clientSecret, expiresIn, err := KiroAuth.ImportFromSsoToken(token, req.Region)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}

		email, userID, _ := KiroAuth.GetUserInfo(accessToken)
		reqData := kiroGoAccountRequest{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			AuthMethod:   "idc",
			Region:       req.Region,
			Enabled:      boolPtr(true),
			ExpiresAt:    time.Now().Unix() + int64(expiresIn),
			Email:        email,
			UserID:       userID,
			MachineID:    uuid.NewString(),
		}
		id, addErr := addOrImportAccount(reqData)
		if addErr != nil {
			errs = append(errs, addErr.Error())
			continue
		}
		imported = append(imported, gin.H{"id": strconv.Itoa(id), "email": email})
	}

	if len(imported) == 0 && len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": strings.Join(errs, "; ")})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "accounts": imported, "errors": errs})
}

func normalizeAuthMethod(authMethod, clientID, clientSecret string) string {
	m := strings.ToLower(strings.TrimSpace(authMethod))
	switch m {
	case "idc", "builderid", "enterprise":
		return "idc"
	case "social", "google", "github":
		return "social"
	default:
		if strings.TrimSpace(clientID) != "" && strings.TrimSpace(clientSecret) != "" {
			return "idc"
		}
		return "social"
	}
}

func AdminImportCredentials(c *gin.Context) {
	var req struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
		ClientID     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
		AuthMethod   string `json:"authMethod"`
		Provider     string `json:"provider"`
		Region       string `json:"region"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refreshToken is required"})
		return
	}

	if strings.TrimSpace(req.Region) == "" {
		req.Region = "us-east-1"
	}
	req.AuthMethod = normalizeAuthMethod(req.AuthMethod, req.ClientID, req.ClientSecret)

	accessToken := strings.TrimSpace(req.AccessToken)
	expiresAt := time.Now().Unix() + 300

	freshAccess, freshRefresh, freshExpires, refreshErr := KiroAuth.RefreshToken(KiroAuth.RefreshInput{
		RefreshToken: strings.TrimSpace(req.RefreshToken),
		ClientID:     strings.TrimSpace(req.ClientID),
		ClientSecret: strings.TrimSpace(req.ClientSecret),
		AuthMethod:   req.AuthMethod,
		Region:       req.Region,
	})
	if refreshErr == nil {
		accessToken = freshAccess
		if strings.TrimSpace(freshRefresh) != "" {
			req.RefreshToken = freshRefresh
		}
		expiresAt = freshExpires
	} else if accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token refresh failed: " + refreshErr.Error()})
		return
	}

	email, userID, _ := KiroAuth.GetUserInfo(accessToken)
	reqData := kiroGoAccountRequest{
		AccessToken:  accessToken,
		RefreshToken: req.RefreshToken,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		AuthMethod:   req.AuthMethod,
		Provider:     req.Provider,
		Region:       req.Region,
		ExpiresAt:    expiresAt,
		Enabled:      boolPtr(true),
		Email:        email,
		UserID:       userID,
		MachineID:    uuid.NewString(),
	}
	id, addErr := addOrImportAccount(reqData)
	if addErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": addErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "account": gin.H{"id": strconv.Itoa(id), "email": email}})
}

func AdminTestAccount(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	account := kiroGoAccountRequest{
		RefreshToken: req.RefreshToken,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		AuthMethod:   normalizeAuthMethod("", req.ClientID, req.ClientSecret),
		Region:       "us-east-1",
	}

	token, _, expiresAt, err := KiroAuth.RefreshToken(KiroAuth.RefreshInput{
		RefreshToken: account.RefreshToken,
		ClientID:     account.ClientID,
		ClientSecret: account.ClientSecret,
		AuthMethod:   account.AuthMethod,
		Region:       account.Region,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	preview := token
	if len(preview) > 12 {
		preview = preview[:6] + "..." + preview[len(preview)-6:]
	}
	c.JSON(http.StatusOK, gin.H{"message": "account token test passed", "access_token_preview": preview, "expires_at": expiresAt})
}

func AdminRefreshTokens(c *gin.Context) {
	refreshed, failed := Utils.RefreshAllActiveTokensNow()
	c.JSON(http.StatusOK, gin.H{"message": "active token refresh completed", "refreshed": refreshed, "failed": failed})
}

func AdminSetRuntimeConfig(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	for k, v := range req {
		vv := ""
		switch t := v.(type) {
		case string:
			vv = t
		case bool:
			if t {
				vv = "true"
			} else {
				vv = "false"
			}
		default:
			b, _ := json.Marshal(t)
			vv = string(b)
		}
		switch k {
		case "bearer_token":
			_ = os.Setenv("BEARER_TOKEN", strings.TrimSpace(vv))
		case "admin_token":
			_ = os.Setenv("ADMIN_TOKEN", strings.TrimSpace(vv))
		case "oidc_url":
			_ = os.Setenv("OIDC_URL", strings.TrimSpace(vv))
		case "amazon_q_url":
			_ = os.Setenv("AMAZON_Q_URL", strings.TrimSpace(vv))
		case "proxy_url":
			_ = os.Setenv("PROXY_URL", strings.TrimSpace(vv))
		case "account_source":
			_ = os.Setenv("ACCOUNT_SOURCE", strings.TrimSpace(vv))
		case "accounts_csv_path":
			_ = os.Setenv("ACCOUNTS_CSV_PATH", strings.TrimSpace(vv))
		case "account_api_url":
			_ = os.Setenv("ACCOUNT_API_URL", strings.TrimSpace(vv))
		case "account_api_token":
			_ = os.Setenv("ACCOUNT_API_TOKEN", strings.TrimSpace(vv))
		case "account_category_id":
			_ = os.Setenv("ACCOUNT_CATEGORY_ID", strings.TrimSpace(vv))
		case "active_token_count":
			_ = os.Setenv("ACTIVE_TOKEN_COUNT", strings.TrimSpace(vv))
		case "max_refresh_attempt":
			_ = os.Setenv("MAX_REFRESH_ATTEMPT", strings.TrimSpace(vv))
		}
	}

	if err := Utils.SaveRuntimeConfigFromEnv(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "runtime config updated"})
}
