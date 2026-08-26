package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxImageRequestBody int64 = 256 << 20

// ImageBodyLimit 限制 Relay 为重试缓存的图片请求体大小。
func ImageBodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		contentEncoding := strings.TrimSpace(strings.ToLower(c.GetHeader("Content-Encoding")))
		if contentEncoding != "" && contentEncoding != "identity" {
			c.AbortWithStatusJSON(http.StatusUnsupportedMediaType, gin.H{
				"error": gin.H{
					"message": "compressed image request bodies are not supported",
					"type":    "invalid_request_error",
				},
			})
			return
		}
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
