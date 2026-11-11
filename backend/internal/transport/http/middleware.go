package http

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware enforces AUTH_API_KEY security contract via header or query parameter.
func AuthMiddleware(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If no key is configured in dev, pass through
		if strings.TrimSpace(expectedKey) == "" {
			c.Next()
			return
		}

		// Allow SSE streams to pass key via query parameter since standard EventSource cannot set custom headers
		queryKey := c.Query("api_key")
		if queryKey != "" && queryKey == expectedKey {
			c.Next()
			return
		}

		// Check X-API-Key header
		headerKey := c.GetHeader("X-API-Key")
		if headerKey != "" && headerKey == expectedKey {
			c.Next()
			return
		}

		// Check Bearer authorization header
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == expectedKey {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error":   "unauthorized",
			"message": "Invalid or missing API authentication key. Provide valid key in 'X-API-Key' header or '?api_key=' parameter.",
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

