package API

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"kilocli2api/Models"
	"kilocli2api/Utils"
)

type adminConfigRequest struct {
	BearerToken       *string `json:"bearer_token"`
	AdminToken        *string `json:"admin_token"`
	OIDCURL           *string `json:"oidc_url"`
	AmazonQURL        *string `json:"amazon_q_url"`
	ProxyURL          *string `json:"proxy_url"`
	AccountAPIURL     *string `json:"account_api_url"`
	AccountAPIToken   *string `json:"account_api_token"`
	AccountCategoryID *string `json:"account_category_id"`
}

type addAccountRequest struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Activate     bool   `json:"activate"`
}

func envOrDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func maskSecret(raw string) string {
	if raw == "" {
		return ""
	}
	if len(raw) <= 8 {
		return "****"
	}
	return raw[:4] + "..." + raw[len(raw)-4:]
}

func AdminPanel(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(adminPanelHTML))
}

func AdminStatus(c *gin.Context) {
	snapshot := Utils.GetAdminSnapshot()
	c.JSON(http.StatusOK, gin.H{
		"snapshot": snapshot,
		"runtime_config": gin.H{
			"port":                envOrDefault("PORT", "4000"),
			"gin_mode":            envOrDefault("GIN_MODE", "release"),
			"account_source":      envOrDefault("ACCOUNT_SOURCE", "csv"),
			"accounts_csv_path":   os.Getenv("ACCOUNTS_CSV_PATH"),
			"oidc_url":            os.Getenv("OIDC_URL"),
			"amazon_q_url":        os.Getenv("AMAZON_Q_URL"),
			"proxy_url":           os.Getenv("PROXY_URL"),
			"account_api_url":     os.Getenv("ACCOUNT_API_URL"),
			"account_api_token":   maskSecret(os.Getenv("ACCOUNT_API_TOKEN")),
			"account_category_id": envOrDefault("ACCOUNT_CATEGORY_ID", "3"),
			"bearer_token":        maskSecret(os.Getenv("BEARER_TOKEN")),
			"admin_token":         maskSecret(envOrDefault("ADMIN_TOKEN", os.Getenv("BEARER_TOKEN"))),
		},
	})
}

func AdminSetRuntimeConfig(c *gin.Context) {
	var req adminConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.BearerToken != nil {
		if err := os.Setenv("BEARER_TOKEN", strings.TrimSpace(*req.BearerToken)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AdminToken != nil {
		if err := os.Setenv("ADMIN_TOKEN", strings.TrimSpace(*req.AdminToken)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.OIDCURL != nil {
		if err := os.Setenv("OIDC_URL", strings.TrimSpace(*req.OIDCURL)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AmazonQURL != nil {
		if err := os.Setenv("AMAZON_Q_URL", strings.TrimSpace(*req.AmazonQURL)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.ProxyURL != nil {
		if err := os.Setenv("PROXY_URL", strings.TrimSpace(*req.ProxyURL)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AccountAPIURL != nil {
		if err := os.Setenv("ACCOUNT_API_URL", strings.TrimSpace(*req.AccountAPIURL)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AccountAPIToken != nil {
		if err := os.Setenv("ACCOUNT_API_TOKEN", strings.TrimSpace(*req.AccountAPIToken)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AccountCategoryID != nil {
		if err := os.Setenv("ACCOUNT_CATEGORY_ID", strings.TrimSpace(*req.AccountCategoryID)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "runtime config updated"})
}

func AdminAddAccount(c *gin.Context) {
	var req addAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	account, err := Utils.AddManualAccount(req.RefreshToken, req.ClientID, req.ClientSecret, req.Activate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "account added", "account": account})
}

func AdminRefreshTokens(c *gin.Context) {
	refreshed, failed := Utils.RefreshAllActiveTokensNow()
	c.JSON(http.StatusOK, gin.H{
		"message":   "active token refresh completed",
		"refreshed": refreshed,
		"failed":    failed,
	})
}

func AdminTestAccount(c *gin.Context) {
	var req addAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	rt := Models.RefreshToken{
		Token:        strings.TrimSpace(req.RefreshToken),
		ClientId:     strings.TrimSpace(req.ClientID),
		ClientSecret: strings.TrimSpace(req.ClientSecret),
	}
	if rt.Token == "" || rt.ClientId == "" || rt.ClientSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token, client_id, client_secret are required"})
		return
	}

	token, err := Utils.GetAccessTokenFromRefreshToken(rt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":              "account token test passed",
		"access_token_preview": maskSecret(token.Token),
		"expires_at":           token.ExpiresAt,
	})
}

const adminPanelHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>kilocli2 管理面板</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 24px; color: #111827; background: #f8fafc; }
    .card { background: white; border: 1px solid #e5e7eb; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
    h1,h2 { margin: 0 0 12px 0; }
    input, textarea, button { font-size: 14px; padding: 8px; margin: 4px 0; width: 100%; box-sizing: border-box; }
    button { cursor: pointer; background: #111827; color: #fff; border: none; border-radius: 8px; }
    button.secondary { background: #4b5563; }
    pre { white-space: pre-wrap; word-break: break-word; background: #0f172a; color: #e2e8f0; padding: 12px; border-radius: 8px; max-height: 320px; overflow: auto; }
    .row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
    .small { color: #6b7280; font-size: 12px; }
  </style>
</head>
<body>
  <h1>kilocli2 管理面板</h1>
  <p class="small">云端部署可直接访问该页面；请在请求头中携带 <code>x-admin-token</code>（或 Authorization Bearer）。</p>

  <div class="card">
    <h2>状态概览</h2>
    <button onclick="loadStatus()">刷新状态</button>
    <pre id="status"></pre>
  </div>

  <div class="card">
    <h2>手动录入账号令牌</h2>
    <div class="row">
      <div><input id="refresh_token" placeholder="refresh_token" /></div>
      <div><input id="client_id" placeholder="client_id" /></div>
    </div>
    <input id="client_secret" placeholder="client_secret" />
    <label><input id="activate" type="checkbox" style="width:auto;"> 添加后立即激活并拉取 access token</label>
    <button onclick="testAccount()" class="secondary">先测试账号</button>
    <button onclick="addAccount()">添加账号</button>
    <pre id="account_result"></pre>
  </div>

  <div class="card">
    <h2>运行时配置（即时生效）</h2>
    <div class="row">
      <div><input id="bearer_token" placeholder="BEARER_TOKEN（可选，留空不改）" /></div>
      <div><input id="admin_token" placeholder="ADMIN_TOKEN（可选，留空不改）" /></div>
    </div>
    <div class="row">
      <div><input id="oidc_url" placeholder="OIDC_URL（可选）" /></div>
      <div><input id="amazon_q_url" placeholder="AMAZON_Q_URL（可选）" /></div>
    </div>
    <div class="row">
      <div><input id="proxy_url" placeholder="PROXY_URL（可选，可留空清空）" /></div>
      <div><input id="account_api_url" placeholder="ACCOUNT_API_URL（可选）" /></div>
    </div>
    <div class="row">
      <div><input id="account_api_token" placeholder="ACCOUNT_API_TOKEN（可选）" /></div>
      <div><input id="account_category_id" placeholder="ACCOUNT_CATEGORY_ID（可选）" /></div>
    </div>
    <button onclick="updateConfig()">更新配置</button>
    <pre id="config_result"></pre>
  </div>

  <div class="card">
    <h2>Token 管理</h2>
    <button onclick="refreshTokens()">手动刷新全部活跃 token</button>
    <pre id="refresh_result"></pre>
  </div>

  <script>
    function tokenHeader() {
      const token = localStorage.getItem("admin_token") || prompt("请输入 x-admin-token");
      if (!token) throw new Error("缺少 admin token");
      localStorage.setItem("admin_token", token);
      return { "Content-Type": "application/json", "x-admin-token": token };
    }

    async function request(url, method = "GET", body = null) {
      const res = await fetch(url, {
        method,
        headers: tokenHeader(),
        body: body ? JSON.stringify(body) : null
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.error || JSON.stringify(data));
      return data;
    }

    async function loadStatus() {
      try {
        const data = await request("/admin/api/status");
        document.getElementById("status").textContent = JSON.stringify(data, null, 2);
      } catch (e) { document.getElementById("status").textContent = e.message; }
    }

    function accountBody() {
      return {
        refresh_token: document.getElementById("refresh_token").value,
        client_id: document.getElementById("client_id").value,
        client_secret: document.getElementById("client_secret").value,
        activate: document.getElementById("activate").checked
      };
    }

    async function testAccount() {
      try {
        const data = await request("/admin/api/accounts/test", "POST", accountBody());
        document.getElementById("account_result").textContent = JSON.stringify(data, null, 2);
      } catch (e) { document.getElementById("account_result").textContent = e.message; }
    }

    async function addAccount() {
      try {
        const data = await request("/admin/api/accounts", "POST", accountBody());
        document.getElementById("account_result").textContent = JSON.stringify(data, null, 2);
        loadStatus();
      } catch (e) { document.getElementById("account_result").textContent = e.message; }
    }

    async function updateConfig() {
      const body = {};
      [
        "bearer_token","admin_token","oidc_url","amazon_q_url",
        "account_api_url","account_api_token","account_category_id"
      ].forEach((k) => {
        const v = document.getElementById(k).value;
        if (v !== "") body[k] = v;
      });
      body.proxy_url = document.getElementById("proxy_url").value;
      try {
        const data = await request("/admin/api/config", "POST", body);
        document.getElementById("config_result").textContent = JSON.stringify(data, null, 2);
        loadStatus();
      } catch (e) { document.getElementById("config_result").textContent = e.message; }
    }

    async function refreshTokens() {
      try {
        const data = await request("/admin/api/tokens/refresh", "POST", {});
        document.getElementById("refresh_result").textContent = JSON.stringify(data, null, 2);
        loadStatus();
      } catch (e) { document.getElementById("refresh_result").textContent = e.message; }
    }

    loadStatus();
  </script>
</body>
</html>`
