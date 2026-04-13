package main

import (
	"kilocli2api/API"
	"kilocli2api/Middleware"

	"github.com/gin-gonic/gin"
)

func setupRouter(r *gin.Engine) {
	v1 := r.Group("/v1")
	v1.Use(Middleware.BearerAuth()) // Apply bearer token authentication
	{
		v1.POST("/chat/completions", API.ChatCompletions)
		v1.POST("/messages", API.Messages)
		v1.POST("/messages/count_tokens", API.CountTokens)
		v1.GET("/models", API.ListModels)
	}

	// Debug endpoint without authentication
	r.POST("/debug/token", API.DebugToken)
	r.POST("/debug/anthropic2q", API.DebugAnthropic2Q)

	r.GET("/admin", API.AdminPanel)
	r.GET("/admin/", API.AdminPanel)

	admin := r.Group("/admin/api")
	admin.Use(Middleware.AdminAuth())
	{
		admin.GET("/status", API.AdminStatus)
		admin.GET("/accounts", API.AdminGetAccounts)
		admin.POST("/config", API.AdminSetRuntimeConfig)
		admin.GET("/settings", API.AdminGetSettings)
		admin.POST("/accounts", API.AdminAddAccount)
		admin.POST("/accounts/batch", API.AdminBatchAccounts)
		admin.PUT("/accounts/:id", API.AdminUpdateAccount)
		admin.DELETE("/accounts/:id", API.AdminDeleteAccount)
		admin.POST("/accounts/test", API.AdminTestAccount)
		admin.POST("/tokens/refresh", API.AdminRefreshTokens)
		admin.GET("/stats", API.AdminGetStats)
		admin.POST("/stats/reset", API.AdminResetStats)
		admin.GET("/endpoint", API.AdminGetEndpoint)
		admin.POST("/endpoint", API.AdminUpdateEndpoint)
		admin.GET("/version", API.AdminVersion)
	}

	r.NoRoute(API.NotFound)
}
