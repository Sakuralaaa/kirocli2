package Middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedToken := os.Getenv("ADMIN_TOKEN")
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
			authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "x-admin-token or Authorization header required"})
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
