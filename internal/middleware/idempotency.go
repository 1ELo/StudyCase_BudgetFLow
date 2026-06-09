package middleware

import (
	"bytes"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type idempotentRequest struct {
	Key          string `gorm:"primaryKey"`
	StatusCode   int
	ResponseBody []byte
}

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Idempotency middleware ensures that requests with the same Idempotency-Key
// header are only processed once. If processed, it caches and returns the result.
func Idempotency(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			c.Next()
			return
		}

		var req idempotentRequest
		err := db.Table("idempotent_requests").Where("key = ?", key).First(&req).Error
		if err == nil {
			// Found cached response
			var jsonData map[string]interface{}
			_ = json.Unmarshal(req.ResponseBody, &jsonData)
			c.JSON(req.StatusCode, jsonData)
			c.Abort()
			return
		}

		// Not found, capture the response
		w := &responseBodyWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = w

		c.Next()

		// Save the response if it's a valid status code (e.g. 2xx, 4xx)
		if c.Writer.Status() >= 200 {
			newReq := idempotentRequest{
				Key:          key,
				StatusCode:   c.Writer.Status(),
				ResponseBody: w.body.Bytes(),
			}
			_ = db.Table("idempotent_requests").Create(&newReq).Error
		}
	}
}
