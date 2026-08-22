package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxImageRequestBody int64 = 256 << 20

// ImageBodyLimit 限制 Relay 为重试缓存的图片请求体大小。
func ImageBodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxImageRequestBody {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"message": "image request body is too large",
					"type":    "invalid_request_error",
				},
			})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxImageRequestBody)
		c.Next()
	}
}
