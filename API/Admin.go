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
	AccountSource     *string `json:"account_source"`
	AccountsCSVPath   *string `json:"accounts_csv_path"`
	AccountAPIURL     *string `json:"account_api_url"`
	AccountAPIToken   *string `json:"account_api_token"`
	AccountCategoryID *string `json:"account_category_id"`
	ActiveTokenCount  *string `json:"active_token_count"`
	MaxRefreshAttempt *string `json:"max_refresh_attempt"`
}

type addAccountRequest struct {
	RefreshToken string `json:"refresh_token"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Activate     bool   `json:"activate"`
}

type updateAccountRequest struct {
	Enabled      *bool   `json:"enabled"`
	RefreshToken *string `json:"refresh_token"`
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
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

func runtimeConfigPayload() gin.H {
	return gin.H{
		"port":                envOrDefault("PORT", "4000"),
		"gin_mode":            envOrDefault("GIN_MODE", "release"),
		"account_source":      envOrDefault("ACCOUNT_SOURCE", "manual"),
		"accounts_csv_path":   os.Getenv("ACCOUNTS_CSV_PATH"),
		"oidc_url":            os.Getenv("OIDC_URL"),
		"amazon_q_url":        os.Getenv("AMAZON_Q_URL"),
		"proxy_url":           os.Getenv("PROXY_URL"),
		"account_api_url":     os.Getenv("ACCOUNT_API_URL"),
		"account_api_token":   maskSecret(os.Getenv("ACCOUNT_API_TOKEN")),
		"account_category_id": envOrDefault("ACCOUNT_CATEGORY_ID", "3"),
		"active_token_count":  envOrDefault("ACTIVE_TOKEN_COUNT", "10"),
		"max_refresh_attempt": envOrDefault("MAX_REFRESH_ATTEMPT", "3"),
		"bearer_token":        maskSecret(os.Getenv("BEARER_TOKEN")),
		"admin_token":         maskSecret(envOrDefault("ADMIN_TOKEN", os.Getenv("BEARER_TOKEN"))),
	}
}

func AdminPanel(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(adminPanelHTML))
}

func AdminStatus(c *gin.Context) {
	snapshot := Utils.GetAdminSnapshot()
	c.JSON(http.StatusOK, gin.H{
		"snapshot":       snapshot,
		"runtime_config": runtimeConfigPayload(),
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
	if req.AccountSource != nil {
		if err := os.Setenv("ACCOUNT_SOURCE", strings.TrimSpace(*req.AccountSource)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.AccountsCSVPath != nil {
		if err := os.Setenv("ACCOUNTS_CSV_PATH", strings.TrimSpace(*req.AccountsCSVPath)); err != nil {
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
	if req.ActiveTokenCount != nil {
		if err := os.Setenv("ACTIVE_TOKEN_COUNT", strings.TrimSpace(*req.ActiveTokenCount)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if req.MaxRefreshAttempt != nil {
		if err := os.Setenv("MAX_REFRESH_ATTEMPT", strings.TrimSpace(*req.MaxRefreshAttempt)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := Utils.SaveRuntimeConfigFromEnv(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

func AdminGetAccounts(c *gin.Context) {
	c.JSON(http.StatusOK, Utils.ListAdminAccounts())
}

func AdminBatchAccounts(c *gin.Context) {
	var req []map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	success, errors := Utils.BatchAddManualAccounts(req)
	c.JSON(http.StatusOK, gin.H{
		"success": success,
		"errors":  errors,
	})
}

func AdminUpdateAccount(c *gin.Context) {
	id, err := Utils.ParseAccountID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req updateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	account, updateErr := Utils.UpdateAccountByID(id, req.Enabled, req.ClientID, req.ClientSecret, req.RefreshToken)
	if updateErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": updateErr.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "account": account})
}

func AdminDeleteAccount(c *gin.Context) {
	id, err := Utils.ParseAccountID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Utils.DeleteAccountByID(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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

func AdminGetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, runtimeConfigPayload())
}

func AdminGetStats(c *gin.Context) {
	snapshot := Utils.GetAdminSnapshot()
	c.JSON(http.StatusOK, gin.H{
		"total_accounts":         snapshot.TotalAccounts,
		"active_accounts":        snapshot.ActiveAccountCount,
		"disabled_accounts":      snapshot.DisabledAccountCount,
		"valid_tokens":           snapshot.ValidTokenCount,
		"account_source":         snapshot.AccountSource,
		"active_token_configured": envOrDefault("ACTIVE_TOKEN_COUNT", "10"),
	})
}

func AdminResetStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "no runtime stats to reset in kirocli mode"})
}

func AdminGetEndpoint(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"oidc_url":     os.Getenv("OIDC_URL"),
		"amazon_q_url": os.Getenv("AMAZON_Q_URL"),
	})
}

func AdminUpdateEndpoint(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if v, ok := req["oidc_url"]; ok {
		if err := os.Setenv("OIDC_URL", strings.TrimSpace(v)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if v, ok := req["amazon_q_url"]; ok {
		if err := os.Setenv("AMAZON_Q_URL", strings.TrimSpace(v)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := Utils.SaveRuntimeConfigFromEnv(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func AdminVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":    "kirocli2",
		"mode":    "kirocli",
		"manager": "kiro-go-style",
	})
}

const adminPanelHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>kirocli2 管理面板</title>
  <style>
    body { margin: 0; font-family: -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif; background: #f3f4f6; color: #111827; }
    .wrap { max-width: 1200px; margin: 0 auto; padding: 20px; }
    .header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; gap: 8px; }
    .btn { border: none; border-radius: 10px; background: #111827; color: #fff; padding: 10px 14px; cursor: pointer; }
    .btn.gray { background: #4b5563; }
    .cards { display: grid; grid-template-columns: repeat(5,minmax(130px,1fr)); gap: 10px; margin-bottom: 16px; }
    .kpi { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 12px; }
    .kpi .n { font-size: 24px; font-weight: 700; margin-top: 6px; }
    .panel { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 14px; margin-bottom: 16px; }
    h2 { margin: 0 0 10px 0; font-size: 18px; }
    .row { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
    .row3 { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; }
    input, textarea, select { width: 100%; box-sizing: border-box; border: 1px solid #d1d5db; border-radius: 8px; padding: 8px; margin-bottom: 8px; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { padding: 8px; border-bottom: 1px solid #e5e7eb; text-align: left; }
    .muted { color: #6b7280; font-size: 12px; }
    .ok { color: #065f46; }
    .bad { color: #991b1b; }
    pre { background: #0f172a; color: #dbeafe; border-radius: 8px; padding: 10px; max-height: 220px; overflow: auto; white-space: pre-wrap; word-break: break-word; }
    @media (max-width: 980px) { .cards { grid-template-columns: 1fr 1fr; } .row, .row3 { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="header">
      <div>
        <h1 style="margin:0;">kirocli2 管理面板</h1>
        <div class="muted">管理逻辑对齐 Kiro-Go；请求链路保持 kirocli。</div>
      </div>
      <div style="display:flex; gap:8px;">
        <button class="btn gray" onclick="resetAuth()">重置口令</button>
        <button class="btn" onclick="refreshAll()">刷新全部</button>
      </div>
    </div>

    <div class="cards">
      <div class="kpi"><div>总账号</div><div class="n" id="k_total">-</div></div>
      <div class="kpi"><div>活跃账号</div><div class="n ok" id="k_active">-</div></div>
      <div class="kpi"><div>禁用账号</div><div class="n bad" id="k_disabled">-</div></div>
      <div class="kpi"><div>有效 Token</div><div class="n" id="k_valid">-</div></div>
      <div class="kpi"><div>账号来源</div><div class="n" style="font-size:16px;" id="k_source">-</div></div>
    </div>

    <div class="panel">
      <h2>账号管理</h2>
      <div class="row3">
        <input id="refresh_token" placeholder="refresh_token" />
        <input id="client_id" placeholder="client_id" />
        <input id="client_secret" placeholder="client_secret" />
      </div>
      <label><input id="activate" type="checkbox" style="width:auto;"> 添加后立即激活</label>
      <div style="display:flex; gap:8px;">
        <button class="btn gray" onclick="testAccount()">先测试</button>
        <button class="btn" onclick="addAccount()">添加账号</button>
      </div>
      <textarea id="batch_accounts" rows="4" placeholder='批量导入(JSON数组): [{"refresh_token":"...","client_id":"...","client_secret":"...","activate":false}]'></textarea>
      <button class="btn gray" onclick="batchAdd()">批量导入</button>
      <pre id="account_result"></pre>
      <div style="overflow:auto; margin-top:8px;">
        <table>
          <thead><tr><th>ID</th><th>Client</th><th>RefreshToken</th><th>状态</th><th>过期时间</th><th>操作</th></tr></thead>
          <tbody id="accounts_tbody"></tbody>
        </table>
      </div>
    </div>

    <div class="panel">
      <h2>运行配置</h2>
      <div class="row">
        <input id="bearer_token" placeholder="BEARER_TOKEN（可选）" />
        <input id="admin_token" placeholder="ADMIN_TOKEN（可选）" />
      </div>
      <div class="row">
        <input id="oidc_url" placeholder="OIDC_URL" />
        <input id="amazon_q_url" placeholder="AMAZON_Q_URL" />
      </div>
      <div class="row">
        <input id="proxy_url" placeholder="PROXY_URL（可留空清空）" />
        <select id="account_source">
          <option value="">ACCOUNT_SOURCE（不修改）</option>
          <option value="manual">manual</option>
          <option value="csv">csv</option>
          <option value="api">api</option>
        </select>
      </div>
      <div class="row">
        <input id="accounts_csv_path" placeholder="ACCOUNTS_CSV_PATH" />
        <input id="account_api_url" placeholder="ACCOUNT_API_URL" />
      </div>
      <div class="row">
        <input id="account_api_token" placeholder="ACCOUNT_API_TOKEN" />
        <input id="account_category_id" placeholder="ACCOUNT_CATEGORY_ID" />
      </div>
      <div class="row">
        <input id="active_token_count" placeholder="ACTIVE_TOKEN_COUNT" />
        <input id="max_refresh_attempt" placeholder="MAX_REFRESH_ATTEMPT" />
      </div>
      <button class="btn" onclick="updateConfig()">保存配置</button>
      <pre id="config_result"></pre>
    </div>

    <div class="panel">
      <h2>Token 操作</h2>
      <button class="btn" onclick="refreshTokens()">手动刷新活跃 token</button>
      <pre id="refresh_result"></pre>
    </div>
  </div>

  <script>
    function adminToken() {
      let token = localStorage.getItem("admin_password");
      if (!token) {
        token = prompt("请输入管理口令（ADMIN_TOKEN）");
      }
      if (!token) throw new Error("缺少管理口令");
      localStorage.setItem("admin_password", token);
      return token;
    }

    function headers() {
      const token = adminToken();
      return { "Content-Type": "application/json", "x-admin-token": token, "X-Admin-Password": token };
    }

    async function req(url, method = "GET", body = null) {
      const res = await fetch(url, { method, headers: headers(), body: body ? JSON.stringify(body) : null });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(data.error || JSON.stringify(data));
      return data;
    }

    function setText(id, value) {
      document.getElementById(id).textContent = typeof value === "string" ? value : JSON.stringify(value, null, 2);
    }

    function accountBody() {
      return {
        refresh_token: document.getElementById("refresh_token").value.trim(),
        client_id: document.getElementById("client_id").value.trim(),
        client_secret: document.getElementById("client_secret").value.trim(),
        activate: document.getElementById("activate").checked
      };
    }

    async function loadStatus() {
      const data = await req("/admin/api/status");
      const s = data.snapshot || {};
      document.getElementById("k_total").textContent = s.total_accounts ?? "-";
      document.getElementById("k_active").textContent = s.active_account_count ?? "-";
      document.getElementById("k_disabled").textContent = s.disabled_account_count ?? "-";
      document.getElementById("k_valid").textContent = s.valid_token_count ?? "-";
      document.getElementById("k_source").textContent = s.account_source ?? "-";
      return data;
    }

    async function loadAccounts() {
      const accounts = await req("/admin/api/accounts");
      const tbody = document.getElementById("accounts_tbody");
      tbody.innerHTML = "";
      accounts.forEach((a) => {
        const accountID = Number(a.id);
        if (!Number.isFinite(accountID) || accountID <= 0) return;
        const tr = document.createElement("tr");
        const expiresDisplay = a.expires_at ? new Date(a.expires_at * 1000).toLocaleString() : "-";
        tr.innerHTML = "<td>" + accountID + "</td>" +
          "<td>" + (a.client_id || "") + "</td>" +
          "<td>" + (a.refresh_token_preview || "") + "</td>" +
          "<td>" + (a.disabled ? "禁用" : (a.active ? "活跃" : "待激活")) + "</td>" +
          "<td>" + expiresDisplay + "</td>" +
          "<td><button class='btn gray' onclick='toggleAccount(" + accountID + "," + (a.disabled ? "true" : "false") + ")'>切换启用</button> <button class='btn gray' onclick='deleteAccount(" + accountID + ")'>删除</button></td>";
        tbody.appendChild(tr);
      });
    }

    async function loadSettings() {
      const s = await req("/admin/api/settings");
      ["oidc_url","amazon_q_url","proxy_url","accounts_csv_path","account_api_url","account_category_id","active_token_count","max_refresh_attempt"].forEach((k) => {
        if (document.getElementById(k)) document.getElementById(k).value = s[k] || "";
      });
      if (s.account_source) document.getElementById("account_source").value = s.account_source;
    }

    async function testAccount() {
      try { setText("account_result", await req("/admin/api/accounts/test", "POST", accountBody())); }
      catch (e) { setText("account_result", e.message); }
    }

    async function addAccount() {
      try {
        setText("account_result", await req("/admin/api/accounts", "POST", accountBody()));
        await refreshAll();
      } catch (e) { setText("account_result", e.message); }
    }

    async function batchAdd() {
      try {
        const raw = document.getElementById("batch_accounts").value.trim();
        const payload = raw ? JSON.parse(raw) : [];
        setText("account_result", await req("/admin/api/accounts/batch", "POST", payload));
        await refreshAll();
      } catch (e) { setText("account_result", e.message); }
    }

    async function toggleAccount(id, disabled) {
      try {
        await req("/admin/api/accounts/" + id, "PUT", { enabled: disabled });
        await refreshAll();
      } catch (e) { setText("account_result", e.message); }
    }

    async function deleteAccount(id) {
      if (!confirm("确认删除账号 #" + id + " ?")) return;
      try {
        await req("/admin/api/accounts/" + id, "DELETE");
        await refreshAll();
      } catch (e) { setText("account_result", e.message); }
    }

    async function updateConfig() {
      const body = {};
      ["bearer_token","admin_token","oidc_url","amazon_q_url","account_source","accounts_csv_path","account_api_url","account_api_token","account_category_id","active_token_count","max_refresh_attempt"].forEach((k) => {
        const v = document.getElementById(k).value;
        if (v !== "") body[k] = v;
      });
      body.proxy_url = document.getElementById("proxy_url").value;
      try {
        setText("config_result", await req("/admin/api/config", "POST", body));
        await refreshAll();
      } catch (e) { setText("config_result", e.message); }
    }

    async function refreshTokens() {
      try {
        setText("refresh_result", await req("/admin/api/tokens/refresh", "POST", {}));
        await refreshAll();
      } catch (e) { setText("refresh_result", e.message); }
    }

    async function refreshAll() {
      try {
        await Promise.all([loadStatus(), loadAccounts(), loadSettings()]);
      } catch (e) {
        setText("account_result", e.message);
      }
    }

    function resetAuth() {
      localStorage.removeItem("admin_password");
      alert("已清除本地管理口令，下次请求会重新输入。");
    }

    refreshAll();
  </script>
</body>
</html>`
