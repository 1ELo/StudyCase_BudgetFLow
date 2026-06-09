package middleware

import (
	"net/http"
	"strings"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/shared/token"
	"github.com/gin-gonic/gin"
)

func Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized", "error": gin.H{"code": "UNAUTHORIZED"}})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized", "error": gin.H{"code": "UNAUTHORIZED"}})
			return
		}
		claims, err := token.ParseToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "message": "unauthorized", "error": gin.H{"code": "UNAUTHORIZED"}})
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("publicID", claims.PublicID)
		c.Set("role", claims.Role)
		c.Next()
	}
}
