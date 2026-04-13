package Middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"kilocli2api/Utils"
)

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedToken := os.Getenv("ADMIN_TOKEN")
		if expected := strings.TrimSpace(Utils.GetKiroGoAdminPassword()); expected != "" {
			expectedToken = expected
		}
		if expectedToken == "" {
			expectedToken = os.Getenv("BEARER_TOKEN")
		}
		if expectedToken == "" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication unavailable",
			})
			c.Abort()
			return
		}

		token := strings.TrimSpace(c.GetHeader("x-admin-token"))
		if token == "" {
			token = strings.TrimSpace(c.GetHeader("X-Admin-Password"))
		}
		if token == "" {
			authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "x-admin-token, X-Admin-Password, or Authorization Bearer header required"})
			c.Abort()
			return
		}

		if token != expectedToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid admin token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
