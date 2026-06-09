package middleware

import (
	"net/http"

	"github.com/1ELo/StudyCase_BudgetFLow/internal/domain"
	"github.com/gin-gonic/gin"
)

func Authorize(roles ...domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "access denied", "error": gin.H{"code": "FORBIDDEN"}})
			return
		}
		userRole := domain.Role(roleVal.(string))
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "message": "access denied", "error": gin.H{"code": "FORBIDDEN"}})
	}
}
