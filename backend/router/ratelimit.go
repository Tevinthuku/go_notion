package router

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"golang.org/x/time/rate"
)

// Global variable to store limiters
var ipLimiters = sync.Map{}

// RateLimitConfig holds the configuration for rate limiting
type RateLimitConfig struct {
	// Requests specifies the number of requests
	Requests int
	// Period specifies the time window for the rate limit
	Period time.Duration
	// Burst specifies the maximum number of requests that can be made at once
	Burst int
}

// IPRateLimiter middleware generator
func IPRateLimiter(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := getRealIP(c)

		// using method + path because we can have the same path for different methods
		routeKey := c.Request.Method + c.FullPath()
		limiterKey := ip + ":" + routeKey

		limiterI, exists := ipLimiters.Load(limiterKey)
		if !exists {
			// interval specifies the time between requests within the same IP and route combination
			// eg: if Period was 1 minute and Requests was 60, then interval would be 1 second and burst would be 5
			// which means we can make 60 requests per minute per IP and route combination
			// however, due to the burst, we can make an additional 5 requests immediately
			interval := config.Period / time.Duration(config.Requests)
			limiter := rate.NewLimiter(rate.Every(interval), config.Burst)
			ipLimiters.Store(limiterKey, limiter)
			limiterI = limiter
		}

		limiter := limiterI.(*rate.Limiter)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// getRealIP attempts to get the real IP address considering proxy headers
// using c.ClientIP() is not enough because the IP address can be spoofed using proxy headers
func getRealIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// Get the first (original) IP in the list
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xrip := c.GetHeader("X-Real-IP"); xrip != "" {
		return xrip
	}

	return c.ClientIP()
}
