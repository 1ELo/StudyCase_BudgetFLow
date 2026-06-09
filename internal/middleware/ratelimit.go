package middleware

import (
	"net/http"
	"sync"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/apperror"
	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// rateLimiterManager stores rate limiters per IP address.
type rateLimiterManager struct {
	limiters sync.Map
	limit    rate.Limit
	burst    int
}

// newRateLimiterManager creates a new manager with the specified rps (requests per second) and burst.
func newRateLimiterManager(rps float64, burst int) *rateLimiterManager {
	return &rateLimiterManager{
		limit: rate.Limit(rps),
		burst: burst,
	}
}

// getLimiter retrieves or creates a rate limiter for the given IP.
func (m *rateLimiterManager) getLimiter(ip string) *rate.Limiter {
	v, exists := m.limiters.Load(ip)
	if !exists {
		limiter := rate.NewLimiter(m.limit, m.burst)
		m.limiters.Store(ip, limiter)
		return limiter
	}
	return v.(*rate.Limiter)
}

// RateLimiter is a Gin middleware that limits requests per IP.
// rps = requests per second, burst = max burst size.
// E.g. RateLimiter(100.0/60.0, 100) allows 100 requests per minute.
func RateLimiter(rps float64, burst int) gin.HandlerFunc {
	manager := newRateLimiterManager(rps, burst)

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		limiter := manager.getLimiter(clientIP)

		if !limiter.Allow() {
			response.Error(c, apperror.NewAppError(http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many requests. Please try again later."))
			c.Abort()
			return
		}
		c.Next()
	}
}
