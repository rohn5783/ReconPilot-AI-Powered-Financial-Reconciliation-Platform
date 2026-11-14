package http

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"reconpilot/internal/service"
)

// AuthMiddleware enforces AUTH_API_KEY security contract or JWT Bearer user session.
func AuthMiddleware(expectedKey string, authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Check Bearer authorization header (JWT token or API key)
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// If it matches expected API key directly:
			if expectedKey != "" && token == expectedKey {
				c.Next()
				return
			}
			// If authSvc is provided, validate JWT
			if authSvc != nil {
				claims, err := authSvc.ValidateToken(token)
				if err == nil && claims != nil {
					c.Set("user_id", claims.UserID)
					c.Set("user_email", claims.Email)
					c.Set("user_role", claims.Role)
					c.Set("user_full_name", claims.FullName)
					c.Next()
					return
				}
			}
		}

		// 2. Allow SSE streams to pass key/token via query parameter since standard EventSource cannot set custom headers
		queryKey := c.Query("api_key")
		if queryKey != "" {
			if expectedKey != "" && queryKey == expectedKey {
				c.Next()
				return
			}
			if authSvc != nil {
				claims, err := authSvc.ValidateToken(queryKey)
				if err == nil && claims != nil {
					c.Set("user_id", claims.UserID)
					c.Set("user_email", claims.Email)
					c.Set("user_role", claims.Role)
					c.Set("user_full_name", claims.FullName)
					c.Next()
					return
				}
			}
		}

		// 3. Check X-API-Key header
		headerKey := c.GetHeader("X-API-Key")
		if headerKey != "" && expectedKey != "" && headerKey == expectedKey {
			c.Next()
			return
		}

		// 4. If no expectedKey configured and in dev mode, pass through
		if strings.TrimSpace(expectedKey) == "" {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid or missing authentication credentials. Provide a valid JWT token in 'Authorization: Bearer <token>' or API key in 'X-API-Key' / '?api_key='.",
		})
	}
}

// CORSMiddleware enables secure cross-origin requests for the React workstation.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With, X-API-Key")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggerMiddleware formats request latency and status.
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[HTTP] %d | %13v | %15s | %-7s %s",
			status,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// RecoveryMiddleware catches panics and renders structured error response.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC RECOVERED] %v", err)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_server_error",
					"message": "An unexpected error occurred in the financial reconciliation pipeline.",
				})
			}
		}()
		c.Next()
	}
}

